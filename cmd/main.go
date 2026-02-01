package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"Final_1/internal/models"
)

// In-memory storage
var (
	mu sync.Mutex

	movies = []models.Movie{
		{ID: 1, Title: "Avatar", Genre: "Sci-Fi", Duration: 162},
		{ID: 2, Title: "Inception", Genre: "Action", Duration: 148},
	}

	tickets      = map[int]models.Ticket{}
	nextTicketID = 1
)

func main() {

	store := NewMovieStore()
	h := NewMovieHandler(store)
	// 3 endpoints
	http.HandleFunc("/movies-list", moviesHandler) // GET
	http.HandleFunc("/book", bookHandler)          // POST
	http.HandleFunc("/ticket", ticketHandler)      // GET ?id=1

	http.HandleFunc("/movies", h.Movies)
	http.HandleFunc("/movies/", h.MovieByID)

	//корень
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK. Use /movies, /book, /ticket?id=1"))
	})

	go func() {
		for {
			time.Sleep(20 * time.Second)
			mu.Lock()
			count := len(tickets)
			mu.Unlock()
			fmt.Printf("[SYSTEM]: Health check OK. Total tickets issued: %d\n", count)
		}
	}()

	fmt.Println("Server running: http://localhost:8080")
	_ = http.ListenAndServe(":8080", nil)
}

// GET /movies
func moviesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(movies)
}

//POST /book

func bookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type BookRequest struct {
		SessionID int `json:"session_id"`
		SeatID    int `json:"seat_id"`
		UserID    int `json:"user_id"`
	}

	var req BookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.SessionID == 0 || req.SeatID == 0 || req.UserID == 0 {
		http.Error(w, "session_id, seat_id, user_id required", http.StatusBadRequest)
		return
	}

	mu.Lock()
	id := nextTicketID
	nextTicketID++

	t := models.Ticket{
		ID:        id,
		SessionID: req.SessionID,
		SeatID:    req.SeatID,
		UserID:    req.UserID,
		Status:    "BOOKED",
	}

	tickets[id] = t
	mu.Unlock()

	_ = time.Now()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)

	go func(ticketID int) {
		time.Sleep(5 * time.Second)
		fmt.Printf("[Async]: Confirmation for ticket #%d has been sent to user email.\n", ticketID)
	}(t.ID)
}

// GET /ticket?id=1
func ticketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "use /ticket?id=1", http.StatusBadRequest)
		return
	}

	mu.Lock()
	t, ok := tickets[id]
	mu.Unlock()

	if !ok {
		http.Error(w, "ticket not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}
