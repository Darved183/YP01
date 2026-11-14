package handlers

import (
	"YP01/internal/database"
	"YP01/internal/models"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type PhotoHandler struct {
	Repo *database.Repository
}

func (h *PhotoHandler) Register(r *mux.Router) {
	r.HandleFunc("/photo", h.Create).Methods("POST")
	r.HandleFunc("/photo/{id}", h.Get).Methods("GET")
	r.HandleFunc("/photo/{id}", h.Delete).Methods("DELETE")
}

func (h *PhotoHandler) Create(w http.ResponseWriter, r *http.Request) {

	err := r.ParseMultipartForm(20 << 20)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid form")
		return
	}

	productID := r.FormValue("product_id")
	if productID == "" {
		writeError(w, http.StatusBadRequest, "product_id is required")
		return
	}

	if _, err = h.Repo.GetProductByID(productID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	photo := models.Photo{
		ID:        uuid.NewString(),
		ProductID: productID,
	}
	photo.Filename = photo.ID + filepath.Ext(header.Filename)

	if err = os.MkdirAll("uploads", 0755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	path := filepath.Join("uploads", photo.Filename)
	dst, err := os.Create(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		os.Remove(path)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = h.Repo.CreatePhoto(&photo); err != nil {
		os.Remove(path)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, photo)
}

func (h *PhotoHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	photo, err := h.Repo.GetPhoto(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "photo not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	path := filepath.Join("uploads", filepath.Base(photo.Filename))
	if _, err = os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	http.ServeFile(w, r, path)
}

func (h *PhotoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	photo, err := h.Repo.GetPhoto(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "photo not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = h.Repo.DeletePhoto(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = os.Remove(filepath.Join("uploads", filepath.Base(photo.Filename)))
	w.WriteHeader(http.StatusNoContent)
}
