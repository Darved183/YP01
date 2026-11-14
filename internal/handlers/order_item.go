package handlers

import (
	"YP01/internal/database"
	"YP01/internal/models"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type OrderItemHandler struct {
	Repo *database.Repository
}

func (h *OrderItemHandler) Register(r *mux.Router) {
	r.HandleFunc("/order_item", h.Create).Methods("POST")
	r.HandleFunc("/order_item/{order_id}", h.List).Methods("GET")
}

func (h *OrderItemHandler) Create(w http.ResponseWriter, r *http.Request) {

	var item models.OrderItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if item.OrderID == "" || item.ProductID == "" {
		writeError(w, http.StatusBadRequest, "order_id and product_id are required")
		return
	}

	if err := h.Repo.CreateOrderItem(&item); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, item)
}

func (h *OrderItemHandler) List(w http.ResponseWriter, r *http.Request) {
	orderID := mux.Vars(r)["order_id"]

	items, err := h.Repo.GetOrderItems(orderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []models.OrderItem{}
	}
	WriteJSON(w, http.StatusOK, items)
}
