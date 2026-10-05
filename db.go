package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func databaseDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		env("DB_USER", "media"),
		env("DB_PASSWORD", "media"),
		env("DB_HOST", "127.0.0.1"),
		env("DB_PORT", "3306"),
		env("DB_NAME", "media_api"),
	)
}

func connect() (*sql.DB, error) {
	db, err := sql.Open("mysql", databaseDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Minute)

	var lastErr error
	for attempt := 1; attempt <= 30; attempt++ {
		lastErr = db.Ping()
		if lastErr == nil {
			return db, nil
		}
		log.Printf("database is not ready (attempt %d): %v", attempt, lastErr)
		time.Sleep(time.Second)
	}
	db.Close()
	return nil, lastErr
}
