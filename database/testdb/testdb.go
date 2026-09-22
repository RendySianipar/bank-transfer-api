// Package testdb provides a shared helper for connecting to the
// integration test database. It lives in its own regular (non-_test.go)
// package deliberately: symbols defined inside a _test.go file are only
// visible within that same package's test binary and cannot be imported
// by other packages' tests. Putting the shared helper here lets both
// database's own tests and Service's integration tests use the exact
// same connection logic, instead of each maintaining a separate,
// possibly inconsistent copy.
//
// Production code (main.go) never imports this package, so none of
// this - or the "testing" package it depends on - ends up in your
// actual deployed server binary.
package testdb

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// Connect opens a connection to the integration test database, using
// credentials from .env.test. t.Fatal is called (failing the calling
// test immediately) if the env file can't be loaded or the connection
// can't be established - callers don't need to check an error return.
func Connect(t *testing.T) *sql.DB {
	t.Helper()

	// godotenv.Load does NOT overwrite variables that are already set
	// in the environment - if TEST_DB_HOST etc. are already exported
	// (e.g. in CI), those take precedence over .env.test's values.
	// The error from Load is intentionally ignored here: if .env.test
	// doesn't exist but the variables are already set some other way
	// (again, e.g. CI secrets), we still want to proceed rather than
	// fail immediately.
	_ = godotenv.Load("../.env.test")

	host := os.Getenv("TEST_DB_HOST")
	port := os.Getenv("TEST_DB_PORT")
	user := os.Getenv("TEST_DB_USER")
	password := os.Getenv("TEST_DB_PASSWORD")
	dbName := os.Getenv("TEST_DB_NAME")

	if dbName == "" {
		t.Fatal("TEST_DB_NAME is not configured - check .env.test exists and is loadable from this package's directory")
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Asia%%2FJakarta",
		user, password, host, port, dbName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("failed to ping test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// Reset wipes all rows from every table used by the app, in an order
// that respects foreign key constraints (children before parents:
// transfers/idempotency_keys reference accounts, accounts references
// users). Call this at the START of each integration test (not the
// end) so a previous test's leftover data - or a prior FAILED test
// that didn't get to clean up - never leaks into the next test.
func Reset(t *testing.T, db *sql.DB) {
	t.Helper()

	tables := []string{"transfers", "idempotency_keys", "accounts", "users"}

	for _, table := range tables {
		if _, err := db.Exec("DELETE FROM " + table); err != nil {
			t.Fatalf("failed to clean table %s: %v", table, err)
		}
	}
}
