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

type CartHandler struct {
	Repo *database.Repository
}

func (h *CartHandler) Register(r *mux.Router) {
	r.HandleFunc("/cart", h.Create).Methods("POST")
	r.HandleFunc("/cart/{user_id}", h.GetByUser).Methods("GET")
	r.HandleFunc("/cart/{id}", h.Update).Methods("PUT")
	r.HandleFunc("/cart/{id}", h.Delete).Methods("DELETE")
}

func (h *CartHandler) Create(w http.ResponseWriter, r *http.Request) {

	var cart models.Cart
	if err := json.NewDecoder(r.Body).Decode(&cart); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if cart.UserID == "" || cart.ProductID == "" {
		writeError(w, http.StatusBadRequest, "user_id and product_id are required")
		return
	}
	if cart.Count < 1 {
		cart.Count = 1
	}

	if err := h.Repo.CreateCart(&cart); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, cart)
}

func (h *CartHandler) GetByUser(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["user_id"]

	carts, err := h.Repo.GetCartByUser(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if carts == nil {
		carts = []models.Cart{}
	}
	WriteJSON(w, http.StatusOK, carts)
}

func (h *CartHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	cart, err := h.Repo.GetCart(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "cart not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = json.NewDecoder(r.Body).Decode(cart); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	cart.ID = id
	if cart.Count < 1 {
		writeError(w, http.StatusBadRequest, "count must be > 0")
		return
	}

	if err = h.Repo.PutCart(cart); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, cart)
}

func (h *CartHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if _, err := h.Repo.GetCart(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "cart not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.Repo.DeleteCart(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
