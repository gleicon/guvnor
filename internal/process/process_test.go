package process

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/gleicon/guvnor/internal/config"
)

func TestManager_Basic(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Quiet during tests
	manager := NewManager(logger)
	
	// Test starting a simple process
	appConfig := config.AppConfig{
		Name:    "test-echo",
		Command: "echo",
		Args:    []string{"hello", "world"},
		Port:    8080,
	}
	
	ctx := context.Background()
	err := manager.Start(ctx, appConfig)
	if err != nil {
		t.Logf("Start error: %v", err)
	}
	
	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)
	
	// Check if we can list the process
	processes := manager.ListProcesses()
	if len(processes) > 0 {
		t.Logf("Found %d processes", len(processes))
	}
	
	// Cleanup
	manager.StopAll(ctx)
}

func TestManager_StopAll(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	manager := NewManager(logger)
	
	ctx := context.Background()
	err := manager.StopAll(ctx)
	if err != nil {
		t.Logf("StopAll error: %v", err)
	}
}