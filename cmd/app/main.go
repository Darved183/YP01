package main

import (
	"YP01/internal/app"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		slog.Error("Error loading .env file", "error", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}
