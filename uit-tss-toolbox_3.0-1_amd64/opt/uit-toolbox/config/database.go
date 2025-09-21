package config

import (
	"database/sql"
	"log"
	"os"
	"time"
)

func NewDBConnection(appConfig AppConfig) (*sql.DB, error) {
	var db *sql.DB
	// Connect to db with pgx
	log.Println("Attempting connection to database...")
	dbConnScheme := "postgres"
	dbConnHost := "127.0.0.1"
	dbConnPort := "5432"
	dbConnUser := "uitweb"
	dbConnDBName := "uitdb"
	dbConnPass := appConfig.UIT_WEB_SVC_PASSWD
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
