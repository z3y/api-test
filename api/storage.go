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

func (s *Storage) Migrate() error {

	fmt.Println("setting up database migrations")
	version, err := s.getVersion()
	if err != nil {
		return err
	}

	fmt.Println("database version: ", version)

	migrations := []func(s *Storage) error{
		func(s *Storage) error {
			query := `create table if not exists account (
				id serial primary key,
				username varchar(255),
				encypted_password varchar(255),
				date_joined timestamp
			)`
			_, err := s.db.Exec(query)
			return err
		},
	}

	previousVersion := version

	for i, f := range migrations {
		if version <= i {
			err := f(s)
			if err != nil {
				return err
			}
			version++
			fmt.Println("database version updated from:", version-1, "to:", version)
		}
	}

	if previousVersion != version {
		if err := s.setVersion(version); err != nil {
			return err
		}
	}

	return err
}
func (s *Storage) setVersion(version int) error {
	_, err := s.db.Exec("update version set version = $1", version)
	return err
}

func (s *Storage) getVersion() (int, error) {
	_, err := s.db.Exec("create table if not exists version(version int)")
	if err != nil {
		return 0, err
	}

	rows, err := s.db.Query("select * from version")
	if err != nil {
		return 0, err
	}

	version := -1
	if rows.Next() {
		err := rows.Scan(
			&version,
		)
		if err != nil {
			return 0, err
		}
	} else {
		_, err := s.db.Query("insert into version(version) values (0)")
		if err != nil {
			return 0, err
		}
		version = 0
	}

	return version, nil
}
