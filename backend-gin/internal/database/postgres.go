package database

import "database/sql"

type Database struct {
	DB *sql.DB
}

func NewDatabase(connectionString string) (*Database, error) {
	//Open database connection
	db, err := sql.Open("postgres", connectionString) //sql driver
	if err != nil {
		return nil, err
	}

	//configure connection pool
	
}