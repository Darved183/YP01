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

type OrderHandler struct {
	Repo *database.Repository
}

func (h *OrderHandler) Register(r *mux.Router) {
	r.HandleFunc("/order", h.Create).Methods("POST")
	r.HandleFunc("/order", h.List).Methods("GET")
	r.HandleFunc("/order/{id}", h.Get).Methods("GET")
	r.HandleFunc("/order/{id}", h.Update).Methods("PUT")
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {

	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if order.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if order.Status == "" {
		order.Status = "Принят"
	}
	if !statusOk(order.Status) {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	if order.DateTime == "" {
		order.DateTime = time.Now().Format("2006-01-02T15:04:05")
	}

	if err := h.Repo.CreateOrder(&order); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, order)
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	orders, err := h.Repo.GetOrders(database.OrderFilter{
		Page:      page,
		Limit:     limit,
		Status:    q.Get("status"),
		StartDate: q.Get("start"),
		EndDate:   q.Get("end"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if orders == nil {
		orders = []models.Order{}
	}
	WriteJSON(w, http.StatusOK, orders)
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	order, err := h.Repo.GetOrder(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	order, err := h.Repo.GetOrder(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = json.NewDecoder(r.Body).Decode(order); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	order.ID = id
	if !statusOk(order.Status) {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}

	if err = h.Repo.PutOrder(order); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, order)
}

func statusOk(s string) bool {
	list := []string{"Принят", "В работе", "Собран", "Готов к выдаче", "Отменён"}
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
