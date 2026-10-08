package main

import(
	"YP01/internal/app"
	"github.com/joho/godotenv"
	"log"
)

func main(){

	if err:= godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	if err:= app.Run(); err != nil {
	    log.Fatal(err)
	}
}