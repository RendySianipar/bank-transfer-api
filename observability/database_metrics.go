package observability

import (
	"database/sql"
	"log/slog"
	"time"
)

// MonitorDatabasePool starts a background goroutine that logs database pool stats
func MonitorDatabasePool(db *sql.DB, logger *slog.Logger, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			stats := db.Stats()

			logger.Info("database pool stats",
				"open_connections", stats.OpenConnections,
				"in_use", stats.InUse,
				"idle", stats.Idle,
				"wait_count", stats.WaitCount,
				"wait_duration_ms", stats.WaitDuration.Milliseconds(),
				"max_idle_closed", stats.MaxIdleClosed,
				"max_lifetime_closed", stats.MaxLifetimeClosed,
			)
		}
	}()
}
