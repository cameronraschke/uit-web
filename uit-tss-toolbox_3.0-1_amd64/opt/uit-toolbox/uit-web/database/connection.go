package database

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"time"
	"uit-toolbox/types"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewDBConnection(dbConnection *types.DBConnection) (*sql.DB, error) {
	if dbConnection == nil {
		return nil, fmt.Errorf("db connection is nil")
	}

	dbConnURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(dbConnection.DBUsername, dbConnection.DBPassword),
		Host:   net.JoinHostPort(dbConnection.DBHost, dbConnection.DBPort),
		Path:   dbConnection.DBName,
	}
	dbConnQuery := dbConnURL.Query()
	dbConnQuery.Set("sslmode", "disable")
	dbConnURL.RawQuery = dbConnQuery.Encode()

	// Open the database connection
	dbConn, err := sql.Open("pgx", dbConnURL.String())
	if err != nil {
		return nil, fmt.Errorf("error opening db: %w", err)
	}

	// Set defaults for dbConn connection
	dbConn.SetMaxOpenConns(50)
	dbConn.SetMaxIdleConns(25)
	dbConn.SetConnMaxIdleTime(1 * time.Hour)
	dbConn.SetConnMaxLifetime(24 * time.Hour)

	// Check if the database connection is valid
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dbConn.PingContext(ctx); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("error pinging db: %w", err)
	}

	return dbConn, nil
}
