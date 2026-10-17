package main

import (
	"YP01/internal/app"
	"log"

	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
