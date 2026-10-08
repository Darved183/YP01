package main

import(
	"YP01/internal/app"
	"YP01/internal/database"
	"github.com/joho/godotenv"
	"log"
)

func main(){

	if err:= godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err = db.Load();
	if err != nil {
	    log.Fatal(err)
	}

	if err:= app.Run(); err != nil {
	    log.Fatal(err)
	}
}