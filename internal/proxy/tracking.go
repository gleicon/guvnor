package proxy

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// generateUUID4 generates a UUID v4 string
func generateUUID4() string {
	// Generate 16 random bytes
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to a timestamp-based UUID if crypto/rand fails
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	
	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant bits
	
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// injectTrackingHeader manages the request tracking header
// This creates a chain of UUIDs separated by semicolons to trace requests across services
func (s *Server) injectTrackingHeader(req *http.Request, r *http.Request) {
	if !s.config.Server.EnableTracking {
		return
	}
	
	headerName := s.config.Server.TrackingHeader
	if headerName == "" {
		headerName = "X-GUVNOR-TRACKING"
	}
	
	// Generate a new UUID for this hop
	newUUID := generateUUID4()
	
	// Check if tracking header already exists
	existingHeader := r.Header.Get(headerName)
	
	var trackingValue string
	if existingHeader != "" {
		// Append new UUID to existing chain with semicolon separator
		trackingValue = fmt.Sprintf("%s;%s", existingHeader, newUUID)
		
		s.logger.WithFields(map[string]interface{}{
			"existing_chain": existingHeader,
			"new_uuid":      newUUID,
			"full_chain":    trackingValue,
		}).Debug("Appending to existing tracking chain")
	} else {
		// First hop in the chain
		trackingValue = newUUID
		
		s.logger.WithFields(map[string]interface{}{
			"new_uuid":    newUUID,
			"header_name": headerName,
		}).Debug("Starting new tracking chain")
	}
	
	// Set the tracking header on the proxied request
	req.Header.Set(headerName, trackingValue)
	
	// Log for debugging and observability
	s.processManager.GetLogManager().Log("proxy-server", "debug", 
		fmt.Sprintf("Request tracking: %s=%s", headerName, trackingValue))
}

// extractTrackingChain extracts and parses the tracking chain from a request
// This can be useful for logging and debugging
func extractTrackingChain(r *http.Request, headerName string) []string {
	if headerName == "" {
		headerName = "X-GUVNOR-TRACKING"
	}
	
	trackingHeader := r.Header.Get(headerName)
	if trackingHeader == "" {
		return nil
	}
	
	// Split by semicolon to get individual UUIDs
	return strings.Split(trackingHeader, ";")
}

// getTrackingInfo returns tracking information for logging purposes
func (s *Server) getTrackingInfo(r *http.Request) map[string]interface{} {
	if !s.config.Server.EnableTracking {
		return nil
	}
	
	headerName := s.config.Server.TrackingHeader
	if headerName == "" {
		headerName = "X-GUVNOR-TRACKING"
	}
	
	chain := extractTrackingChain(r, headerName)
	if len(chain) == 0 {
		return nil
	}
	
	info := map[string]interface{}{
		"tracking_header": headerName,
		"tracking_chain":  strings.Join(chain, ";"),
		"hop_count":       len(chain),
	}
	
	// Add first and last UUID for easy identification
	if len(chain) > 0 {
		info["first_uuid"] = chain[0]
		info["last_uuid"] = chain[len(chain)-1]
	}
	
	return info
}