package app

import (
	"YP01/internal/config"
	"YP01/internal/database"
	"encoding/json"

	"log/slog"
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

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	db, err := database.Load()
	if err != nil {
		slog.Error("Error loading .env file", "error", err)
		os.Exit(1)
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
	slog.Info("Server is listening", "addr", Config.Addr, "components", components)
	return http.ListenAndServe(Config.Addr, router)
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
