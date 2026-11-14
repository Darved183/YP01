package handlers

import (
	"YP01/internal/database"
	"YP01/internal/models"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type UserHandler struct {
	Repo *database.Repository
}

func (h *UserHandler) Register(r *mux.Router) {
	r.HandleFunc("/user", h.Create).Methods("POST")
	r.HandleFunc("/user", h.List).Methods("GET")
	// этот путь должен быть выше /user/{id}, иначе role воспримется как id
	r.HandleFunc("/user/role/{id}", h.ByRole).Methods("GET")
	r.HandleFunc("/user/{id}", h.Update).Methods("PUT")
	r.HandleFunc("/user/{id}", h.Delete).Methods("DELETE")
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if user.Name == "" || user.Email == "" || user.RoleID == "" {
		writeError(w, http.StatusBadRequest, "name, email and role_id are required")
		return
	}
	if user.RegistrationDate == "" {
		user.RegistrationDate = time.Now().Format("2006-01-02T15:04:05")
	}

	if err := h.Repo.CreateUser(&user); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	users, err := h.Repo.GetUsers(database.UserFilter{
		Page:  page,
		Limit: limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if users == nil {
		users = []models.User{}
	}
	WriteJSON(w, http.StatusOK, users)
}

func (h *UserHandler) ByRole(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	users, err := h.Repo.GetUsersByRole(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if users == nil {
		users = []models.User{}
	}
	WriteJSON(w, http.StatusOK, users)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	user, err := h.Repo.GetUser(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = json.NewDecoder(r.Body).Decode(user); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	user.ID = id

	if err = h.Repo.PutUser(user); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, user)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if _, err := h.Repo.GetUser(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.Repo.DeleteUser(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
