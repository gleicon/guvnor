package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gleicon/guvnor/internal/api"
	"github.com/gleicon/guvnor/internal/logs"
	"github.com/gleicon/guvnor/internal/process"
)

// Client handles communication with the running guvnor server
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient creates a new API client
func NewClient(httpPort int) *Client {
	mgmtPort := api.GetManagementPort(httpPort)
	
	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", mgmtPort),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsServerRunning checks if the guvnor server is running
func (c *Client) IsServerRunning() bool {
	resp, err := c.client.Get(c.baseURL + "/api/ping")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

// GetStatus gets the current process status
func (c *Client) GetStatus() ([]process.ProcessInfo, error) {
	resp, err := c.client.Get(c.baseURL + "/api/status")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to guvnor server: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	var response struct {
		Processes []process.ProcessInfo `json:"processes"`
		Count     int                   `json:"count"`
		Timestamp string                `json:"timestamp"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return response.Processes, nil
}

// GetLogs gets logs from the server
func (c *Client) GetLogs(processName string, lines int) ([]logs.LogEntry, error) {
	url := c.baseURL + "/api/logs"
	if processName != "" {
		url = fmt.Sprintf("%s/%s", url, processName)
	}
	
	if lines > 0 {
		url = fmt.Sprintf("%s?lines=%d", url, lines)
	}
	
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to guvnor server: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	var response struct {
		Logs      []logs.LogEntry `json:"logs"`
		Count     int             `json:"count"`
		Process   string          `json:"process"`
		Lines     int             `json:"lines"`
		Timestamp string          `json:"timestamp"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return response.Logs, nil
}

// StreamLogs streams logs from the server using Server-Sent Events
func (c *Client) StreamLogs(processName string, callback func([]logs.LogEntry)) error {
	url := c.baseURL + "/api/logs/stream"
	if processName != "" {
		url = fmt.Sprintf("%s?process=%s", url, processName)
	}
	
	resp, err := c.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to guvnor server: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	// Parse Server-Sent Events
	reader := NewSSEReader(resp.Body)
	
	for {
		event, err := reader.ReadEvent()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("error reading event stream: %w", err)
		}
		
		var data struct {
			Type      string          `json:"type"`
			Logs      []logs.LogEntry `json:"logs,omitempty"`
			Count     int             `json:"count,omitempty"`
			Timestamp string          `json:"timestamp"`
		}
		
		if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
			continue // Skip invalid events
		}
		
		if data.Type == "logs" && len(data.Logs) > 0 {
			callback(data.Logs)
		}
	}
}

// StopProcesses stops all processes
func (c *Client) StopProcesses() ([]process.StopResult, error) {
	resp, err := c.client.Post(c.baseURL+"/api/stop", "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to guvnor server: %w", err)
	}
	defer resp.Body.Close()
	
	var response struct {
		Results   []process.StopResult `json:"results"`
		Success   bool                 `json:"success"`
		Error     string               `json:"error,omitempty"`
		Timestamp string               `json:"timestamp"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	if !response.Success && response.Error != "" {
		return response.Results, fmt.Errorf("server error: %s", response.Error)
	}
	
	return response.Results, nil
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Type string
	Data string
}

// SSEReader reads Server-Sent Events from an io.Reader
type SSEReader struct {
	reader io.Reader
}

// NewSSEReader creates a new SSE reader
func NewSSEReader(r io.Reader) *SSEReader {
	return &SSEReader{reader: r}
}

// ReadEvent reads the next Server-Sent Event
func (r *SSEReader) ReadEvent() (*SSEEvent, error) {
	var buf bytes.Buffer
	temp := make([]byte, 1)
	
	event := &SSEEvent{}
	
	for {
		n, err := r.reader.Read(temp)
		if err != nil {
			return nil, err
		}
		
		if n > 0 {
			if temp[0] == '\n' {
				line := buf.String()
				buf.Reset()
				
				if line == "" {
					// Empty line indicates end of event
					if event.Data != "" {
						return event, nil
					}
					continue
				}
				
				if strings.HasPrefix(line, "data: ") {
					event.Data = strings.TrimPrefix(line, "data: ")
				} else if strings.HasPrefix(line, "event: ") {
					event.Type = strings.TrimPrefix(line, "event: ")
				}
				// Ignore other SSE fields for now (id, retry, etc.)
				
			} else if temp[0] != '\r' {
				buf.WriteByte(temp[0])
			}
		}
	}
}

// DetectServerPort tries to detect which port the guvnor server is running on
func DetectServerPort() (int, error) {
	// Try common ports
	commonPorts := []int{8081, 8080, 8090, 3000}
	
	for _, port := range commonPorts {
		client := NewClient(port)
		if client.IsServerRunning() {
			return port, nil
		}
	}
	
	return 0, fmt.Errorf("no running guvnor server found on common ports")
}