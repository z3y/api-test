package main

import (
	"database/sql"
	"flag"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type Storage struct {
	db *sql.DB
}

func (s *Storage) Connect() error {
	fmt.Println("connecting to database")

	local := flag.Bool("local", false, "local db")
	flag.Parse()

	var connectionString string
	if *local {
		connectionString = "host=localhost port=5432 user=postgres dbname=postgres password=" + pgPassword + " sslmode=disable"
	} else {
		connectionString = "host=db_postgres port=5432 user=postgres dbname=postgres password=" + pgPassword + " sslmode=disable"
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
	username          string
	uuid              uuid.UUID
	dateJoined        time.Time
	encryptedPassword string
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

func (s *Storage) NewUser(username, password string) (*User, error) {

	taken, err := s.UsernameTaken(username)
	if err != nil {
		return nil, err
	}

	if taken {
		return nil, fmt.Errorf("username taken")
	}

	encryptedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	usr := User{
		username:          username,
		encryptedPassword: string(encryptedPassword),
		uuid:              uuid.New(),
		dateJoined:        time.Now().UTC(),
	}

	fmt.Println("create user", usr.username, usr.uuid)

	query := `insert into account
	(username, encrypted_password, date_joined, uuid)
	values ($1, $2, $3, $4)`

	_, err = s.db.Exec(
		query,
		usr.username,
		usr.encryptedPassword,
		usr.dateJoined,
		usr.uuid.String(),
	)

	if err != nil {
		return nil, err
	}

	return &usr, nil
}

func (s *Storage) DeleteUser(uuid string) error {

	fmt.Println("delete user", uuid)

	_, err := s.db.Exec("delete from account where uuid = $1", uuid)
	return err
}

func (s *Storage) GetUserByUuid(uuidStr string) (*User, error) {

	uuid, err := uuid.Parse(uuidStr)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query("select username, date_joined, encrypted_password from account where uuid = $1", uuidStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	user := new(User)
	if rows.Next() {
		rows.Scan(&user.username, &user.dateJoined, &user.encryptedPassword)
	} else {
		return nil, fmt.Errorf("user not found")
	}
	user.uuid = uuid

	return user, nil
}

func PasswordValid(hashedPassword, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

func (s *Storage) LoginValid(username, password string) (bool, string) {

	rows, err := s.db.Query("select encrypted_password, uuid from account where username = $1", username)
	if err != nil {
		return false, ""
	}
	defer rows.Close()

	var encryptedPassword string
	var id string
	if rows.Next() {
		rows.Scan(&encryptedPassword, &id)
	} else {
		return false, ""
	}

	if PasswordValid(encryptedPassword, password) {
		return true, id
	}

	return false, ""
}
