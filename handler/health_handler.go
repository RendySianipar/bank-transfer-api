package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// LivenessHandler checks if the application process is alive
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "alive",
	})
}

// ReadinessHandler checks if the application is ready to accept traffic
// It verifies that the database connection is working
func ReadinessHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Create a timeout context (2 seconds max)
		ctx, cancel := context.WithTimeout(
			r.Context(),
			2*time.Second,
		)
		defer cancel()

		// Try to ping the database
		if err := db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "not_ready",
				"reason": "database unreachable",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
		})
	}
}
