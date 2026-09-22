package database

import (
	"bank-transfer-api/database/testdb"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func TestIntegration_DatabaseConnection(t *testing.T) {
	db := testdb.Connect(t)

	if err := db.Ping(); err != nil {
		t.Fatalf("database ping failed: %v", err)
	}

}
