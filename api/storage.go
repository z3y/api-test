package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func (s *Storage) Connect() error {
	fmt.Println("connecting to database")

	pw := os.Getenv("POSTGRES_PASSWORD")
	connectionString := "user=postgres dbname=postgres password=" + pw + " sslmode=disable"
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	s.db = db
	return nil
}

func (s *Storage) SetupSchema() error {
	fmt.Println("setting up database schema")

	query := `create table if not exists account()`

	s.createColumnIfNotExists("account", "id serial primary key")
	s.createColumnIfNotExists("account", "username varchar(255)")
	s.createColumnIfNotExists("account", "encrypted_password varchar(255)")
	s.createColumnIfNotExists("account", "date_joined timestamp")

	_, err := s.db.Exec(query)
	return err
}

func (s *Storage) createColumnIfNotExists(table, column string) error {
	query := fmt.Sprintf("alter table %s add column if not exists %s", table, column)
	_, err := s.db.Exec(query)
	return err
}
