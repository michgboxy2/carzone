package driver

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func InitDB() {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	fmt.Println("Waiting for the database startup...")
	time.Sleep(5 * time.Second)

	var err error
	db, err = sql.Open("postgres", connStr) // Ensure the driver name is "postgres"

	if err != nil {
		log.Fatalf("Error Opening database: %v", err)
		return // Exit the function if there is an error
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error Connecting to the database: %v", err)
		return // Exit the function if there is an error
	}

	fmt.Println("Successfully connected to the database")
}

func GetDB() *sql.DB {
	if db == nil {
		log.Println("Warning: Attempting to access a nil database connection.")
		return nil // Return nil if the database connection is not initialized
	}
	return db // Return the initialized database connection
}

func CloseDB() {
	if db != nil {
		if err := db.Close(); err != nil {
			log.Fatalf("Error Closing The Database: %v", err)
		}
	} else {
		log.Println("Database connection is nil, nothing to close.")
	}
}
