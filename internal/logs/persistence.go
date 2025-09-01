package logs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// PersistentLogManager extends LogManager with file-based persistence
type PersistentLogManager struct {
	*LogManager
	logFile string
}

// NewPersistentLogManager creates a log manager that persists to disk
func NewPersistentLogManager(capacity int, logFile string) *PersistentLogManager {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		// Fallback to temp directory if we can't create the preferred location
		logFile = filepath.Join(os.TempDir(), "guvnor-logs.json")
	}

	plm := &PersistentLogManager{
		LogManager: NewLogManager(capacity),
		logFile:    logFile,
	}

	// Load existing logs
	plm.loadFromFile()

	return plm
}

// Log adds a log entry and persists it
func (plm *PersistentLogManager) Log(process, level, message string) {
	plm.LogManager.Log(process, level, message)
	plm.saveToFile()
}

// saveToFile saves current logs to file
func (plm *PersistentLogManager) saveToFile() {
	// Get all logs from all processes
	allLogs := plm.GetAllLogs(0) // 0 means get all logs
	
	// Convert to JSON
	data, err := json.Marshal(allLogs)
	if err != nil {
		return // Fail silently for now
	}
	
	// Write to temp file first, then rename (atomic operation)
	tmpFile := plm.logFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return
	}
	
	// Atomic rename
	os.Rename(tmpFile, plm.logFile)
}

// loadFromFile loads logs from file
func (plm *PersistentLogManager) loadFromFile() {
	data, err := os.ReadFile(plm.logFile)
	if err != nil {
		return // File doesn't exist or can't be read, that's OK
	}
	
	var logs []LogEntry
	if err := json.Unmarshal(data, &logs); err != nil {
		return // Invalid JSON, ignore and start fresh
	}
	
	// Add logs back to buffers
	for _, entry := range logs {
		// Skip saving to file during load to avoid recursion
		plm.LogManager.Log(entry.Process, entry.Level, entry.Message)
	}
}

// GetLogFile returns the path to the log file
func (plm *PersistentLogManager) GetLogFile() string {
	return plm.logFile
}

// SharedLogEntry represents a log entry for sharing between processes
type SharedLogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Process   string    `json:"process"`
	Message   string    `json:"message"`
	PID       int       `json:"pid,omitempty"`
}

// GetSharedLogFile returns the path to the shared log file
func GetSharedLogFile() string {
	// Use a standard location that all guvnor processes can access
	logDir := filepath.Join(os.TempDir(), "guvnor")
	os.MkdirAll(logDir, 0755)
	return filepath.Join(logDir, "guvnor-shared.log")
}

// WriteSharedLog writes a single log entry to the shared log file
func WriteSharedLog(process, level, message string) {
	logFile := GetSharedLogFile()
	
	entry := SharedLogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Process:   process,
		Message:   message,
		PID:       os.Getpid(),
	}
	
	// Open file in append mode
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	
	// Write JSON entry + newline
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	
	file.Write(data)
	file.WriteString("\n")
}

// ReadSharedLogs reads all log entries from the shared log file
func ReadSharedLogs(maxLines int) ([]LogEntry, error) {
	logFile := GetSharedLogFile()
	
	data, err := os.ReadFile(logFile)
	if err != nil {
		return nil, err
	}
	
	var entries []LogEntry
	lines := splitByNewlines(data)
	
	// Process lines in reverse order to get most recent first
	start := 0
	if maxLines > 0 && len(lines) > maxLines {
		start = len(lines) - maxLines
	}
	
	for i := start; i < len(lines); i++ {
		line := lines[i]
		if len(line) == 0 {
			continue
		}
		
		var sharedEntry SharedLogEntry
		if err := json.Unmarshal(line, &sharedEntry); err != nil {
			continue // Skip invalid lines
		}
		
		entries = append(entries, LogEntry{
			Timestamp: sharedEntry.Timestamp,
			Level:     sharedEntry.Level,
			Process:   sharedEntry.Process,
			Message:   sharedEntry.Message,
		})
	}
	
	return entries, nil
}

// ReadSharedLogsForProcess reads log entries for a specific process
func ReadSharedLogsForProcess(process string, maxLines int) ([]LogEntry, error) {
	allLogs, err := ReadSharedLogs(0) // Get all logs first
	if err != nil {
		return nil, err
	}
	
	var processLogs []LogEntry
	for _, entry := range allLogs {
		if entry.Process == process {
			processLogs = append(processLogs, entry)
		}
	}
	
	// Return last N entries
	if maxLines > 0 && len(processLogs) > maxLines {
		processLogs = processLogs[len(processLogs)-maxLines:]
	}
	
	return processLogs, nil
}

// CleanupSharedLogs removes old shared log files
func CleanupSharedLogs() {
	logFile := GetSharedLogFile()
	os.Remove(logFile)
}

// Helper function to split byte data by newlines
func splitByNewlines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	
	// Add last line if it doesn't end with newline
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	
	return lines
}