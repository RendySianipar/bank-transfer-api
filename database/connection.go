package database

import (
	"database/sql"
	"fmt"
	"os"

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

	// dsn := "root:drillbit@tcp(127.0.0.1:3306)/bank_transfer_db"
	// db, err := sql.Open("mysql", dsn)
	// if err != nil {
	// 	return nil, err
	// }

	// if err := db.Ping(); err != nil {
	// 	return nil, err
	// }

	return db, nil
}
