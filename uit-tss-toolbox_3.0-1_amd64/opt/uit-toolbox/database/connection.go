package database

import (
	"database/sql"
	"os"
	"time"
	config "uit-toolbox/config"
	log "uit-toolbox/logger"
)

func NewDBConnection(appConfig config.AppConfig) (*sql.DB, error) {
	var db *sql.DB
	// Connect to db with pgx
	log.Info("Attempting connection to database...")
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
		log.Error("Unable to connect to database: \n" + dbConnErr.Error())
		os.Exit(1)
	}
	defer db.Close()

	// Check if the database connection is valid
	if err := db.Ping(); err != nil {
		log.Error("Cannot ping database: \n" + err.Error())
		return nil, err
	}
	log.Info("Connected to database successfully")

	// Set defaults for db connection
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(10)
	db.SetConnMaxIdleTime(1 * time.Minute)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}
