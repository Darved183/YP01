package app

import (
	"YP01/internal/config"
	"YP01/internal/database"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

type Status struct {
	Name   string `json:"name"`
	Health bool   `json:"health"`
	Error  string `json:"error"`
}

func Run() error {

	db, err := database.Load()
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	Config := config.Load()

	router := mux.NewRouter()

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status, components := Health(db)
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(components)
	}).Methods("GET")
	_, components := Health(db)
	log.Println("Server is listening: " + Config.Addr)
	json.NewEncoder(os.Stdout).Encode(components)
	return http.ListenAndServe(":8181", router)
}

func Health(db *sqlx.DB) (int, map[string]interface{}) {

	status := 200
	components := []Status{}

	if err := db.Ping(); err != nil {
		components = append(components, Status{
			Name:   "database",
			Health: false,
			Error:  err.Error(),
		})
		status = http.StatusRequestTimeout
	} else {
		components = append(components, Status{
			Name:   "database",
			Health: true,
		})
	}

	resp := map[string]interface{}{
		"status":     status,
		"components": components,
	}
	return status, resp

}
