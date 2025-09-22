package database

import (
	"database/sql"
	"log"
	"os"
	"time"
)

func NewDBConnection(dbName string, dbHost string, dbPort string, dbUsername string, dbPassword string) (*sql.DB, error) {
	var db *sql.DB
	// Connect to db with pgx
	log.Println("Attempting connection to database...")
	dbConnScheme := "postgres"
	dbConnHost := dbHost
	dbConnPort := dbPort
	dbConnUser := dbUsername
	dbConnDBName := dbName
	dbConnPass := dbPassword
	dbConnString := dbConnScheme + "://" + dbConnUser + ":" + dbConnPass + "@" + dbConnHost + ":" + dbConnPort + "/" + dbConnDBName + "?sslmode=disable"
	var dbConnErr error
	db, dbConnErr = sql.Open("pgx", dbConnString)
	if dbConnErr != nil {
		log.Println("Unable to connect to database: \n" + dbConnErr.Error())
		os.Exit(1)
	}
	defer db.Close()

	// Check if the database connection is valid
	if err := db.Ping(); err != nil {
		log.Printf("%s %s", "Cannot ping database:", err.Error())
		return nil, err
	}
	log.Println("Connected to database successfully")

	// Set defaults for db connection
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(10)
	db.SetConnMaxIdleTime(1 * time.Minute)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}
