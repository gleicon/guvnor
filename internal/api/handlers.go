package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gleicon/guvnor/internal/logs"
	"github.com/gleicon/guvnor/internal/process"
)

// Server handles the management API
type Server struct {
	logger         *logrus.Entry
	processManager *process.EnhancedManager
	logManager     *logs.LogManager
	port           int
	server         *http.Server
}

// NewServer creates a new management API server
func NewServer(logger *logrus.Logger, processManager *process.EnhancedManager, logManager *logs.LogManager, port int) *Server {
	return &Server{
		logger:         logger.WithField("component", "api-server"),
		processManager: processManager,
		logManager:     logManager,
		port:           port,
	}
}

// Start starts the management API server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	
	// API routes
	mux.HandleFunc("/api/ping", s.handlePing)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/logs/", s.handleLogsProcess) // For /api/logs/{process}
	mux.HandleFunc("/api/logs/stream", s.handleLogsStream)
	mux.HandleFunc("/api/stop", s.handleStop)
	
	// Add CORS headers for local development
	corsHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			h.ServeHTTP(w, r)
		})
	}

	s.server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", s.port),
		Handler: corsHandler(mux),
	}

	s.logger.WithField("port", s.port).Info("Starting management API server")
	
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Error("Management API server error")
		}
	}()

	return nil
}

// Stop stops the management API server
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	s.logger.Info("Stopping management API server")
	return s.server.Shutdown(ctx)
}

// handlePing handles ping requests for health checking
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.jsonResponse(w, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleStatus handles process status requests
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := s.processManager.GetRunningProcessInfo()
	s.jsonResponse(w, map[string]interface{}{
		"processes": info,
		"count":     len(info),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// handleLogs handles log requests
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	lines := 100 // default
	if l := r.URL.Query().Get("lines"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			lines = parsed
		}
	}

	process := r.URL.Query().Get("process")

	var entries []logs.LogEntry
	if process != "" {
		entries = s.logManager.GetProcessLogs(process, lines)
	} else {
		entries = s.logManager.GetAllLogs(lines)
	}

	s.jsonResponse(w, map[string]interface{}{
		"logs":      entries,
		"count":     len(entries),
		"process":   process,
		"lines":     lines,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// handleLogsProcess handles log requests for specific processes via URL path
func (s *Server) handleLogsProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract process name from path /api/logs/{process}
	path := strings.TrimPrefix(r.URL.Path, "/api/logs/")
	if path == "" || path == "stream" {
		http.Error(w, "Process name required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	lines := 100
	if l := r.URL.Query().Get("lines"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			lines = parsed
		}
	}

	entries := s.logManager.GetProcessLogs(path, lines)
	s.jsonResponse(w, map[string]interface{}{
		"logs":      entries,
		"count":     len(entries),
		"process":   path,
		"lines":     lines,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// handleLogsStream handles streaming logs via Server-Sent Events
func (s *Server) handleLogsStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set up Server-Sent Events
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	process := r.URL.Query().Get("process")
	
	// Get current log count to track new entries
	var lastCount int
	if process != "" {
		lastCount = len(s.logManager.GetProcessLogs(process, 1000))
	} else {
		lastCount = len(s.logManager.GetAllLogs(1000))
	}

	// Send initial data
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"timestamp\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
	w.(http.Flusher).Flush()

	// Poll for new logs
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var newEntries []logs.LogEntry

			if process != "" {
				allEntries := s.logManager.GetProcessLogs(process, 1000)
				if len(allEntries) > lastCount {
					newEntries = allEntries[lastCount:]
					lastCount = len(allEntries)
				}
			} else {
				allEntries := s.logManager.GetAllLogs(1000)
				if len(allEntries) > lastCount {
					newEntries = allEntries[lastCount:]
					lastCount = len(allEntries)
				}
			}

			if len(newEntries) > 0 {
				data := map[string]interface{}{
					"type":      "logs",
					"logs":      newEntries,
					"count":     len(newEntries),
					"timestamp": time.Now().Format(time.RFC3339),
				}

				jsonData, _ := json.Marshal(data)
				fmt.Fprintf(w, "data: %s\n\n", jsonData)
				w.(http.Flusher).Flush()
			}
		}
	}
}

// handleStop handles process stop requests
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := s.processManager.StopAllWithResults(ctx)
	
	response := map[string]interface{}{
		"results":   results,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if err != nil {
		response["error"] = err.Error()
		response["success"] = false
	} else {
		response["success"] = true
	}

	s.jsonResponse(w, response)
}

// jsonResponse sends a JSON response
func (s *Server) jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.WithError(err).Error("Failed to encode JSON response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GetManagementPort calculates the management port from HTTP port
func GetManagementPort(httpPort int) int {
	return httpPort + 1000 // Use +1000 to avoid conflicts
}