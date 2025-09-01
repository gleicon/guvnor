package logs

import "sync"

var (
	globalLogManager *LogManager
	globalOnce       sync.Once
)

// GetGlobalLogManager returns a singleton log manager instance
func GetGlobalLogManager() *LogManager {
	globalOnce.Do(func() {
		globalLogManager = NewLogManager(1000) // 1000 entries per process
	})
	return globalLogManager
}

// SetGlobalLogManager sets the global log manager (for testing or custom configuration)
func SetGlobalLogManager(lm *LogManager) {
	globalLogManager = lm
}