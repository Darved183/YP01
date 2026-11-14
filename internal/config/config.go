package config

import (
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
)

type Config struct {
	Addr string
}

type Status struct {
	Name   string `json:"name"`
	Health bool   `json:"health"`
	Error  string `json:"error"`
}

func Load() Config {

	return Config{
		Addr: os.Getenv("HOST") + ":" + os.Getenv("PORT"),
	}
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
