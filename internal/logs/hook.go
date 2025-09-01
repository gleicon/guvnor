package logs

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// LogManagerHook is a logrus hook that sends logs to the circular buffer
type LogManagerHook struct {
	logManager *LogManager
}

// NewLogManagerHook creates a new hook for logrus that writes to the log manager
func NewLogManagerHook(logManager *LogManager) *LogManagerHook {
	return &LogManagerHook{
		logManager: logManager,
	}
}

// Levels returns the logrus levels this hook will fire for
func (hook *LogManagerHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

// Fire is called when a log entry is made
func (hook *LogManagerHook) Fire(entry *logrus.Entry) error {
	// Extract process name from the fields
	processName := "system"
	
	// First check for app name (more specific)
	if app, exists := entry.Data["app"]; exists {
		processName = fmt.Sprintf("%v", app)
	} else if component, exists := entry.Data["component"]; exists {
		// Use component if no app specified
		processName = fmt.Sprintf("%v", component)
	}
	
	// Get the log level
	level := entry.Level.String()
	
	// Format the message with fields
	message := entry.Message
	if len(entry.Data) > 0 {
		var fields []string
		for key, value := range entry.Data {
			// Skip special fields we already processed, but keep some important ones
			if key == "component" || key == "app" {
				continue
			}
			// Include important fields in the message
			if key == "pid" || key == "port" || key == "mode" || key == "error" {
				fields = append(fields, fmt.Sprintf("%s=%v", key, value))
			}
		}
		if len(fields) > 0 {
			message = fmt.Sprintf("%s (%s)", message, strings.Join(fields, " "))
		}
	}
	
	// Add to log manager
	hook.logManager.Log(processName, level, message)
	
	return nil
}