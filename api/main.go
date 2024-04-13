package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	secretKey []byte
)

func main() {

	key, _ := GenerateRandomAuthKey()
	secretKey = []byte(key)

	fmt.Println("key: ", key)

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

	// _ = storage.NewUser(&User{
	// 	username: "docker",
	// 	password: "hunter2",
	// })
	// storage.DeleteUser("afa62c86-252b-42fe-9a71-5c214974ec77")

	api := NewApi(":3000", storage)
	api.Run()
}
