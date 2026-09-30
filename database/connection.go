package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func Connect() (*sql.DB, error) {

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Asia%%2FJakarta",
		user,
		password,
		host,
		port,
		dbName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)                  // Maximum number of open connections
	db.SetMaxIdleConns(10)                  // Maximum number of idle connections
	db.SetConnMaxLifetime(30 * time.Minute) // How long a connection can be used
	db.SetConnMaxIdleTime(5 * time.Minute)  // How long a connection can be idle before being closed

	return db, nil
}
