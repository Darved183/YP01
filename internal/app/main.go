package app

import (
	"YP01/internal/config"
	"YP01/internal/database"
	"YP01/internal/handlers"
	"YP01/internal/pages"

	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func Run() error {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	db, err := database.Load()
	if err != nil {
		slog.Error("Error loading .env file", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	router := mux.NewRouter()

	Config := config.Load()

	repo := database.NewRepository(db)

	productH := &handlers.ProductHandler{Repo: repo}
	productH.Register(router)

	photoH := &handlers.PhotoHandler{Repo: repo}
	photoH.Register(router)

	roleH := &handlers.RoleHandler{Repo: repo}
	roleH.Register(router)

	userH := &handlers.UserHandler{Repo: repo}
	userH.Register(router)

	cartH := &handlers.CartHandler{Repo: repo}
	cartH.Register(router)

	orderH := &handlers.OrderHandler{Repo: repo}
	orderH.Register(router)

	itemH := &handlers.OrderItemHandler{Repo: repo}
	itemH.Register(router)

	web, err := pages.New(repo)
	if err != nil {
		return err
	}
	web.Register(router)
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("frontend/css"))))
	router.NotFoundHandler = http.HandlerFunc(web.NotFoundPage)

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		status, components := config.Health(db)
		handlers.WriteJSON(w, status, components)

	}).Methods("GET")

	_, components := config.Health(db)
	slog.Info("Server is listening", "addr", Config.Addr, "components", components)
	return http.ListenAndServe(Config.Addr, router)
}
