package main

import (
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {

	id := uuid.New()
	fmt.Println(id.String())

	_, isDocker := os.LookupEnv("POSTGRES_PASSWORD")
	if !isDocker {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	storage := new(Storage)

	if err := storage.Connect(); err != nil {
		log.Fatal(err)
	}

	if err := storage.Migrate(); err != nil {
		log.Fatal(err)
	}

	err := storage.NewUser(&User{
		username: "docker",
		password: "hunter2",
	})

	if err != nil {
		log.Fatal(err)
	}

	// storage.DeleteUser("b62f2146-25dd-498f-9bac-1291a0abcb96")

	// server := NewAPIServer(":3000", store)
	// server.Run()
}
