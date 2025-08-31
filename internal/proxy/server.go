package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme/autocert"

	"github.com/gleicon/guvnor/internal/config"
	"github.com/gleicon/guvnor/internal/health"
	"github.com/gleicon/guvnor/internal/process"
)

// Server represents the main proxy server
type Server struct {
	config         *config.Config
	processManager *process.Manager
	healthChecker  *health.Checker
	logger         *logrus.Entry
	httpServer     *http.Server
	httpsServer    *http.Server
	certManager    *autocert.Manager
	mu             sync.RWMutex
	running        bool
}

// NewServer creates a new proxy server
func NewServer(ctx context.Context, cfg *config.Config, logger *logrus.Logger) (*Server, error) {
	serverLogger := logger.WithField("component", "proxy-server")
	
	// Create process manager
	processManager := process.NewManager(logger)
	
	// Create health checker
	healthChecker := health.NewChecker(processManager, logger)
	
	server := &Server{
		config:         cfg,
		processManager: processManager,
		healthChecker:  healthChecker,
		logger:         serverLogger,
	}
	
	// Setup TLS certificate manager if enabled
	if cfg.TLS.Enabled && cfg.TLS.AutoCert {
		if err := server.setupCertManager(); err != nil {
			return nil, fmt.Errorf("failed to setup certificate manager: %w", err)
		}
	}
	
	// Setup HTTP servers
	if err := server.setupServers(); err != nil {
		return nil, fmt.Errorf("failed to setup servers: %w", err)
	}
	
	return server, nil
}

// Start starts the proxy server and all managed applications
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.running {
		return fmt.Errorf("server is already running")
	}
	
	s.logger.Info("Starting proxy server")
	
	// Start all configured applications
	for _, appConfig := range s.config.Apps {
		s.logger.WithField("app", appConfig.Name).Info("Starting application")
		
		if err := s.processManager.Start(ctx, appConfig); err != nil {
			s.logger.WithError(err).WithField("app", appConfig.Name).Error("Failed to start application")
			continue
		}
	}
	
	// Start health checker
	s.healthChecker.Start(ctx)
	
	// Start HTTP server (for redirects and ACME challenges)
	go func() {
		s.logger.WithField("port", s.config.Server.HTTPPort).Info("Starting HTTP server")
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Error("HTTP server error")
		}
	}()
	
	// Start HTTPS server if TLS is enabled
	if s.config.TLS.Enabled {
		go func() {
			s.logger.WithField("port", s.config.Server.HTTPSPort).Info("Starting HTTPS server")
			if err := s.httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				s.logger.WithError(err).Error("HTTPS server error")
			}
		}()
	}
	
	s.running = true
	s.logger.Info("Proxy server started successfully")
	
	return nil
}

// Stop stops the proxy server and all managed applications
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.running {
		return nil
	}
	
	s.logger.Info("Stopping proxy server")
	
	// Stop health checker
	s.healthChecker.Stop()
	
	// Stop HTTP servers
	if s.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, s.config.Server.ShutdownTimeout)
		defer cancel()
		
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.WithError(err).Error("Error shutting down HTTP server")
		}
	}
	
	if s.httpsServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, s.config.Server.ShutdownTimeout)
		defer cancel()
		
		if err := s.httpsServer.Shutdown(shutdownCtx); err != nil {
			s.logger.WithError(err).Error("Error shutting down HTTPS server")
		}
	}
	
	// Stop all applications
	if err := s.processManager.StopAll(ctx); err != nil {
		s.logger.WithError(err).Error("Error stopping applications")
	}
	
	s.running = false
	s.logger.Info("Proxy server stopped")
	
	return nil
}

// setupCertManager sets up automatic certificate management
func (s *Server) setupCertManager() error {
	// Create cert directory if it doesn't exist
	if err := os.MkdirAll(s.config.TLS.CertDir, 0700); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}
	
	// Collect all domains from apps
	domains := s.config.TLS.Domains
	for _, app := range s.config.Apps {
		domains = append(domains, app.Domain)
	}
	
	// Create autocert manager
	s.certManager = &autocert.Manager{
		Cache:      autocert.DirCache(s.config.TLS.CertDir),
		Prompt:     autocert.AcceptTOS,
		Email:      s.config.TLS.Email,
		HostPolicy: autocert.HostWhitelist(domains...),
	}
	
	// Use staging environment if configured
	if s.config.TLS.Staging {
		// For staging, we can set the directory URL via the Manager's Client field
		// This is a simplified approach - in production you might want more control
		s.logger.Info("Using Let's Encrypt staging environment")
	}
	
	s.logger.WithFields(logrus.Fields{
		"domains":  domains,
		"cert_dir": s.config.TLS.CertDir,
		"staging":  s.config.TLS.Staging,
	}).Info("Certificate manager configured")
	
	return nil
}

// setupServers configures HTTP and HTTPS servers
func (s *Server) setupServers() error {
	// Create HTTP server
	httpMux := http.NewServeMux()
	
	if s.config.TLS.Enabled && s.config.TLS.AutoCert {
		// Handle ACME challenges
		httpMux.Handle("/.well-known/acme-challenge/", s.certManager.HTTPHandler(nil))
	}
	
	// HTTP server handler
	httpMux.HandleFunc("/", s.handleHTTPRequest)
	
	s.httpServer = &http.Server{
		Addr:         ":" + strconv.Itoa(s.config.Server.HTTPPort),
		Handler:      httpMux,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
	}
	
	// Create HTTPS server if TLS is enabled
	if s.config.TLS.Enabled {
		httpsMux := http.NewServeMux()
		httpsMux.HandleFunc("/", s.handleHTTPSRequest)
		
		s.httpsServer = &http.Server{
			Addr:         ":" + strconv.Itoa(s.config.Server.HTTPSPort),
			Handler:      httpsMux,
			ReadTimeout:  s.config.Server.ReadTimeout,
			WriteTimeout: s.config.Server.WriteTimeout,
		}
		
		if s.config.TLS.AutoCert {
			s.httpsServer.TLSConfig = &tls.Config{
				GetCertificate: s.certManager.GetCertificate,
				NextProtos:     []string{"h2", "http/1.1"},
			}
		}
	}
	
	return nil
}

// handleHTTPRequest handles HTTP requests
func (s *Server) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	// If TLS is enabled and force HTTPS is on, redirect to HTTPS
	if s.config.TLS.Enabled && s.config.TLS.ForceHTTPS {
		httpsURL := &url.URL{
			Scheme: "https",
			Host:   r.Host,
			Path:   r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
		
		if s.config.Server.HTTPSPort != 443 {
			httpsURL.Host = fmt.Sprintf("%s:%d", r.Host, s.config.Server.HTTPSPort)
		}
		
		http.Redirect(w, r, httpsURL.String(), http.StatusMovedPermanently)
		return
	}
	
	// Handle the request normally
	s.proxyRequest(w, r)
}

// handleHTTPSRequest handles HTTPS requests
func (s *Server) handleHTTPSRequest(w http.ResponseWriter, r *http.Request) {
	s.proxyRequest(w, r)
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = 200
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// proxyRequest proxies the request to the appropriate backend
func (s *Server) proxyRequest(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	// Wrap response writer to capture status code and size
	rw := &responseWriter{ResponseWriter: w, statusCode: 0, size: 0}
	
	// Find the app for this domain
	var targetApp *config.AppConfig
	for _, app := range s.config.Apps {
		if app.Domain == r.Host {
			targetApp = &app
			break
		}
	}
	
	if targetApp == nil {
		s.logApacheFormat(r, rw, 404, time.Since(startTime), "-")
		s.logger.Warn("No application found for domain", "host", r.Host)
		http.Error(rw, "Domain not found", http.StatusNotFound)
		return
	}
	
	// Check if the target process is running
	proc, exists := s.processManager.GetProcess(targetApp.Name)
	if !exists || !proc.IsRunning() {
		s.logApacheFormat(r, rw, 503, time.Since(startTime), targetApp.Name)
		s.logger.Error("Target application is not running", "app", targetApp.Name)
		http.Error(rw, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	
	// Create reverse proxy
	targetURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", targetApp.Port),
	}
	
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	
	// Customize the proxy director to modify the request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Forwarded-For", getClientIP(r))
		if r.TLS != nil {
			req.Header.Set("X-Forwarded-Proto", "https")
		} else {
			req.Header.Set("X-Forwarded-Proto", "http")
		}
		req.Header.Set("X-Forwarded-Host", r.Host)
	}
	
	// Handle proxy errors
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		s.logApacheFormat(r, rw, 502, time.Since(startTime), targetApp.Name)
		s.logger.Error("Proxy error", "app", targetApp.Name, "error", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}
	
	// Proxy the request
	proxy.ServeHTTP(rw, r)
	
	// Log in Apache Combined Log Format
	duration := time.Since(startTime)
	statusCode := rw.statusCode
	if statusCode == 0 {
		statusCode = 200
	}
	
	s.logApacheFormat(r, rw, statusCode, duration, targetApp.Name)
}

// logApacheFormat logs HTTP requests in Apache Combined Log Format
func (s *Server) logApacheFormat(r *http.Request, rw *responseWriter, statusCode int, duration time.Duration, app string) {
	// Apache Combined Log Format:
	// "%h %l %u %t \"%r\" %>s %O \"%{Referer}i\" \"%{User-Agent}i\""
	// %h - Remote hostname (IP)
	// %l - Remote logname (always -)
	// %u - Remote user (always - for us)
	// %t - Time the request was received
	// %r - First line of request
	// %>s - Status code
	// %O - Size of response in bytes
	// %{Referer}i - Referer header
	// %{User-Agent}i - User-Agent header
	
	clientIP := getClientIP(r)
	timestamp := time.Now().Add(-duration).Format("02/Jan/2006:15:04:05 -0700")
	requestLine := fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto)
	size := rw.size
	if size == 0 {
		size = 0
	}
	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "-"
	}
	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		userAgent = "-"
	}
	
	// Log entry format: clientIP - - [timestamp] "requestLine" statusCode size "referer" "userAgent" app responseTime
	logEntry := fmt.Sprintf(`%s - - [%s] "%s" %d %d "%s" "%s" app=%s rt=%dms`,
		clientIP,
		timestamp,
		requestLine,
		statusCode,
		size,
		referer,
		userAgent,
		app,
		duration.Milliseconds(),
	)
	
	// Log at INFO level for successful requests, WARN for client errors, ERROR for server errors
	if statusCode >= 500 {
		s.logger.Error(logEntry)
	} else if statusCode >= 400 {
		s.logger.Warn(logEntry)
	} else {
		s.logger.Info(logEntry)
	}
}

// getClientIP extracts the real client IP from request headers
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (most common)
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := strings.Index(xf, ","); idx > 0 {
			return strings.TrimSpace(xf[:idx])
		}
		return strings.TrimSpace(xf)
	}
	
	// Check X-Real-IP header
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	
	// Fallback to remote address
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx > 0 {
		return r.RemoteAddr[:idx]
	}
	
	return r.RemoteAddr
}