package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func (s *Storage) Connect() error {
	fmt.Println("connecting to database")

	pw := os.Getenv("POSTGRES_PASSWORD")

	local := flag.Bool("local", false, "local db")
	flag.Parse()

	var connectionString string
	if *local {
		connectionString = "host=localhost port=5432 user=postgres dbname=postgres password=" + pw + " sslmode=disable"
	} else {
		connectionString = "host=db_postgres port=5432 user=postgres dbname=postgres password=" + pw + " sslmode=disable"
	}

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

		// version 1
		func(s *Storage) error {
			query := `
			create table if not exists account(
			id serial primary key,
			username varchar(255),
			encypted_password varchar(255),
			date_joined timestamp
			)`
			_, err := s.db.Exec(query)
			return err
		},

		// version 2
		func(s *Storage) error {
			query := `
			alter table account
			add uuid varchar(255)`
			_, err := s.db.Exec(query)
			return err
		},

		// version 3
		func(s *Storage) error {
			query := `
			alter table account
			rename column encypted_password to encrypted_password`
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
	defer rows.Close()

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

type User struct {
	username   string
	uuid       uuid.UUID
	dateJoined time.Time
	password   string
}

func (s *Storage) UsernameTaken(username string) (bool, error) {
	rows, err := s.db.Query("select count(1) from account where username = $1", username)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	count := 0
	if rows.Next() {
		rows.Scan(&count)
	}

	return count > 0, nil
}

func (s *Storage) NewUser(usr *User) error {

	taken, err := s.UsernameTaken(usr.username)
	if err != nil {
		return err
	}

	if taken {
		return fmt.Errorf("username taken")
	}

	usr.dateJoined = time.Now().UTC()
	usr.uuid = uuid.New()

	fmt.Println("create user", usr.username, usr.uuid)

	query := `insert into account
	(username, encrypted_password, date_joined, uuid)
	values ($1, $2, $3, $4)`

	_, err2 := s.db.Exec(
		query,
		usr.username,
		usr.password,
		usr.dateJoined,
		usr.uuid.String(),
	)

	if err2 != nil {
		return err2
	}

	return nil
}

func (s *Storage) DeleteUser(uuid string) error {

	fmt.Println("delete user", uuid)

	_, err := s.db.Exec("delete from account where uuid = $1", uuid)
	return err
}
