package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// New creates a new structured logger with the specified debug level
func New(debug bool) *logrus.Logger {
	logger := logrus.New()
	
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})
	
	if debug {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
	
	return logger
}

// WithComponent returns a logger with a component field set
func WithComponent(log *logrus.Logger, component string) *logrus.Entry {
	return log.WithField("component", component)
}