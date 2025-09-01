package logs

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Process   string    `json:"process"`
	Message   string    `json:"message"`
}

// CircularBuffer implements a thread-safe circular buffer for log entries
type CircularBuffer struct {
	buffer []LogEntry
	head   int
	tail   int
	size   int
	full   bool
	mu     sync.RWMutex
}

// NewCircularBuffer creates a new circular buffer with the specified capacity
func NewCircularBuffer(capacity int) *CircularBuffer {
	return &CircularBuffer{
		buffer: make([]LogEntry, capacity),
		size:   capacity,
	}
}

// Add adds a new log entry to the buffer
func (cb *CircularBuffer) Add(entry LogEntry) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.buffer[cb.tail] = entry
	cb.tail = (cb.tail + 1) % cb.size
	
	if cb.full {
		cb.head = (cb.head + 1) % cb.size
	}
	
	if cb.tail == cb.head {
		cb.full = true
	}
}

// GetLast returns the last n log entries
func (cb *CircularBuffer) GetLast(n int) []LogEntry {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	if n <= 0 {
		return []LogEntry{}
	}
	
	var entries []LogEntry
	count := cb.count()
	
	if count == 0 {
		return entries
	}
	
	// Limit n to available entries
	if n > count {
		n = count
	}
	
	// Calculate starting position
	start := cb.head
	if cb.full {
		// If buffer is full, start from the oldest entry we want to keep
		start = (cb.tail - n + cb.size) % cb.size
	} else {
		// If buffer is not full, start from beginning or n entries back
		if n > cb.tail {
			start = 0
		} else {
			start = cb.tail - n
		}
	}
	
	// Collect entries
	for i := 0; i < n; i++ {
		pos := (start + i) % cb.size
		entries = append(entries, cb.buffer[pos])
	}
	
	return entries
}

// GetAll returns all log entries in chronological order
func (cb *CircularBuffer) GetAll() []LogEntry {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	count := cb.count()
	if count == 0 {
		return []LogEntry{}
	}
	
	entries := make([]LogEntry, count)
	
	for i := 0; i < count; i++ {
		pos := (cb.head + i) % cb.size
		entries[i] = cb.buffer[pos]
	}
	
	return entries
}

// count returns the number of entries in the buffer (must be called with lock held)
func (cb *CircularBuffer) count() int {
	if cb.full {
		return cb.size
	}
	return cb.tail
}

// Clear removes all entries from the buffer
func (cb *CircularBuffer) Clear() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.head = 0
	cb.tail = 0
	cb.full = false
}

// FormatEntry formats a log entry for display
func FormatEntry(entry LogEntry) string {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	level := strings.ToUpper(entry.Level)
	
	// Color coding for levels (ANSI colors)
	var colorCode string
	switch strings.ToLower(entry.Level) {
	case "error":
		colorCode = "\033[31m" // Red
	case "warn", "warning":
		colorCode = "\033[33m" // Yellow
	case "info":
		colorCode = "\033[32m" // Green
	case "debug":
		colorCode = "\033[36m" // Cyan
	default:
		colorCode = "\033[0m" // Default
	}
	
	resetColor := "\033[0m"
	
	return fmt.Sprintf("%s [%s%s%s] [%s] %s",
		timestamp,
		colorCode,
		level,
		resetColor,
		entry.Process,
		entry.Message,
	)
}

// LogManager manages logs for all processes
type LogManager struct {
	buffers map[string]*CircularBuffer
	mu      sync.RWMutex
	capacity int
}

// NewLogManager creates a new log manager
func NewLogManager(capacity int) *LogManager {
	return &LogManager{
		buffers:  make(map[string]*CircularBuffer),
		capacity: capacity,
	}
}

// Log adds a log entry for a specific process
func (lm *LogManager) Log(process, level, message string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	if _, exists := lm.buffers[process]; !exists {
		lm.buffers[process] = NewCircularBuffer(lm.capacity)
	}
	
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Process:   process,
		Message:   message,
	}
	
	lm.buffers[process].Add(entry)
}

// GetProcessLogs returns the last n log entries for a specific process
func (lm *LogManager) GetProcessLogs(process string, n int) []LogEntry {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	if buffer, exists := lm.buffers[process]; exists {
		return buffer.GetLast(n)
	}
	
	return []LogEntry{}
}

// GetAllLogs returns logs from all processes, interleaved by timestamp
func (lm *LogManager) GetAllLogs(n int) []LogEntry {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	var allEntries []LogEntry
	
	// Collect all entries from all processes
	for _, buffer := range lm.buffers {
		entries := buffer.GetAll()
		allEntries = append(allEntries, entries...)
	}
	
	// Sort by timestamp (bubble sort for simplicity, could use sort.Slice)
	for i := 0; i < len(allEntries)-1; i++ {
		for j := 0; j < len(allEntries)-i-1; j++ {
			if allEntries[j].Timestamp.After(allEntries[j+1].Timestamp) {
				allEntries[j], allEntries[j+1] = allEntries[j+1], allEntries[j]
			}
		}
	}
	
	// Return last n entries
	if n > 0 && n < len(allEntries) {
		return allEntries[len(allEntries)-n:]
	}
	
	return allEntries
}

// GetProcessNames returns all process names that have logs
func (lm *LogManager) GetProcessNames() []string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	var names []string
	for name := range lm.buffers {
		names = append(names, name)
	}
	
	return names
}

// Clear clears logs for a specific process or all processes
func (lm *LogManager) Clear(process string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	if process == "" {
		// Clear all
		for _, buffer := range lm.buffers {
			buffer.Clear()
		}
	} else if buffer, exists := lm.buffers[process]; exists {
		buffer.Clear()
	}
}