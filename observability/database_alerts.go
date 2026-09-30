package observability

import (
	"database/sql"
	"log/slog"
	"time"
)

// AlertOnDatabasePoolProblems checks for unhealthy database pool conditions
func AlertOnDatabasePoolProblems(db *sql.DB, logger *slog.Logger, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		var previousWaitCount int64

		for range ticker.C {
			stats := db.Stats()

			// Alert 1: Too many connections in use
			if stats.InUse > 20 {
				logger.Warn("HIGH DATABASE LOAD",
					"in_use", stats.InUse,
					"max_open", stats.OpenConnections,
				)
			}

			// Alert 2: Requests are waiting for connections
			if stats.WaitCount > previousWaitCount {
				waitsSinceLastCheck := stats.WaitCount - int64(previousWaitCount)
				logger.Warn("REQUESTS WAITING FOR DATABASE CONNECTION",
					"new_waits", waitsSinceLastCheck,
					"total_wait_count", stats.WaitCount,
					"avg_wait_ms", stats.WaitDuration.Milliseconds() / stats.WaitCount,
				)
			}

			// Alert 3: Connection pool is nearly full
			if stats.InUse > stats.OpenConnections-3 {
				logger.Warn("DATABASE CONNECTION POOL NEARLY FULL",
					"in_use", stats.InUse,
					"available", stats.OpenConnections-stats.InUse,
					"max_open", stats.OpenConnections,
				)
			}

			previousWaitCount = stats.WaitCount
		}
	}()
}