package handlers

import (
	"YP01/internal/database"
	"YP01/internal/models"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type ProductHandler struct {
	Repo *database.Repository
}

func (h *ProductHandler) Register(r *mux.Router) {
	r.HandleFunc("/product", h.Create).Methods("POST")
	r.HandleFunc("/product", h.List).Methods("GET")
	r.HandleFunc("/product/{id}", h.Get).Methods("GET")
	r.HandleFunc("/product/{id}", h.Update).Methods("PUT")
	r.HandleFunc("/product/{id}", h.Delete).Methods("DELETE")
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {

	var product models.Product

	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.Repo.CreateProduct(&product); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, product)

}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	f := database.ProductFilter{
		Page:   page,
		Limit:  limit,
		Type:   q.Get("type"),
		Gender: q.Get("gender"),
		Size:   q.Get("size"),
		Color:  q.Get("color"),
		Brand:  q.Get("brand"),
		Season: q.Get("season"),
		Sort:   q.Get("sort"),
		Order:  q.Get("order"),
	}

	products, err := h.Repo.GetProducts(f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, products)
}

func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	product, err := h.Repo.GetProductByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, product)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	product, err := h.Repo.GetProductByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := json.NewDecoder(r.Body).Decode(product); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	product.ID = id
	if err := h.Repo.PutProduct(product); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, product)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	n, err := h.Repo.DeleteProduct(id)

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
