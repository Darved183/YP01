package app

import (
	"fmt"
	"net/http"
	"github.com/gorilla/mux"
	"YP01/internal/config"
)

func Run() error {

	Config := config.Load()
	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request){
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")
	http.Handle("/",router)
    fmt.Println("Server is listening: "+ Config.Addr)
	return http.ListenAndServe(":8181", nil)
}
