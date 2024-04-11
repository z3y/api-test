package main

import (
	"log"

	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	storage := new(Storage)

	if err := storage.Connect(); err != nil {
		log.Fatal(err)
	}

	if err := storage.Migrate(); err != nil {
		log.Fatal(err)
	}

	// server := NewAPIServer(":3000", store)
	// server.Run()
}
