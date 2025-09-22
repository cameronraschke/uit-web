package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewDBConnection(dbName string, dbHost string, dbPort string, dbUsername string, dbPassword string) (*sql.DB, error) {
	log.Println("Attempting connection to database...")
	dbConnURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(dbUsername, dbPassword),
		Host:   net.JoinHostPort(dbHost, dbPort),
		Path:   dbName,
	}
	dbConnQuery := dbConnURL.Query()
	dbConnQuery.Set("sslmode", "disable")
	dbConnURL.RawQuery = dbConnQuery.Encode()

	// Open the database connection
	dbConn, err := sql.Open("pgx", dbConnURL.String())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Set defaults for dbConn connection
	dbConn.SetMaxOpenConns(30)
	dbConn.SetMaxIdleConns(10)
	dbConn.SetConnMaxIdleTime(1 * time.Minute)
	dbConn.SetConnMaxLifetime(5 * time.Minute)

	// Check if the database connection is valid
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dbConn.PingContext(ctx); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	log.Println("Connected to database successfully")

	return dbConn, nil
}
