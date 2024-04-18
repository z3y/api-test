package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	secretKey  []byte
	pgPassword string
)

func main() {

	key, _ := GenerateRandomAuthKey()
	secretKey = []byte(key)

	fmt.Println("key: ", key)

	isDocker := false
	pgPassword, isDocker = os.LookupEnv("POSTGRES_PASSWORD")
	env := make(map[string]string)
	if !isDocker {
		file, err := os.Open(".env")
		if err != nil {
			log.Fatal("error loading .env file")
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()

			parts := strings.SplitN(line, "=", 2)
			env[parts[0]] = parts[1]
		}

		pgPassword = env["POSTGRES_PASSWORD"]
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
