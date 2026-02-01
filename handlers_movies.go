package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type MovieHandler struct {
	store *MovieStore
}

func NewMovieHandler(store *MovieStore) *MovieHandler {
	return &MovieHandler{store: store}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (h *MovieHandler) Movies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, h.store.List())
		return

	case http.MethodPost:
		var m Movie
		if err := readJSON(r, &m); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}

		created, err := h.store.Create(m)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, created)
		return

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
}

func (h *MovieHandler) MovieByID(w http.ResponseWriter, r *http.Request) {
	// expected: /movies/{id}
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] != "movies" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	id, err := strconv.Atoi(parts[1])
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		m, ok := h.store.Get(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "movie not found"})
			return
		}
		writeJSON(w, http.StatusOK, m)
		return

	case http.MethodPatch:
		var p MoviePatch
		if err := readJSON(r, &p); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		updated, err := h.store.Update(id, p)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, updated)
		return

	case http.MethodDelete:
		if !h.store.Delete(id) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "movie not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		return

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
}
