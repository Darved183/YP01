package handlers

import (
	"YP01/internal/database"
	"YP01/internal/models"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
)

type RoleHandler struct {
	Repo *database.Repository
}

func (h *RoleHandler) Register(r *mux.Router) {
	r.HandleFunc("/role", h.Create).Methods("POST")
	r.HandleFunc("/role", h.List).Methods("GET")
	r.HandleFunc("/role/{id}", h.Update).Methods("PUT")
	r.HandleFunc("/role/{id}", h.Delete).Methods("DELETE")
}

func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {

	var role models.Role
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if role.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := h.Repo.CreateRole(&role); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, role)
}

func (h *RoleHandler) List(w http.ResponseWriter, r *http.Request) {
	roles, err := h.Repo.GetRoles()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if roles == nil {
		roles = []models.Role{}
	}
	WriteJSON(w, http.StatusOK, roles)
}

func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	role, err := h.Repo.GetRole(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "role not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = json.NewDecoder(r.Body).Decode(role); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	role.ID = id
	if role.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err = h.Repo.PutRole(role); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, role)
}

func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if _, err := h.Repo.GetRole(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "role not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.Repo.DeleteRole(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
