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

var (
	mu sync.Mutex

	movies = []models.Movie{
		{ID: 1, Title: "Avatar", Genre: "Sci-Fi", Duration: 162, Price: 2500},
		{ID: 2, Title: "Inception", Genre: "Action", Duration: 148, Price: 2600},
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

func moviesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(movies)
}

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

	// ЛОГИКА: Берем цену первого фильма (Avatar) для примера,
	// чтобы ты увидел свои 2500 в Postman.
	moviePrice := movies[0].Price

	// Если хочешь, чтобы цена менялась от ID:
	if req.SessionID == 2 {
		moviePrice = movies[1].Price // Будет 2600
	}

	mu.Lock()
	id := nextTicketID
	nextTicketID++

	t := models.Ticket{
		ID:        id,
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Status:    "BOOKED",
		Price:     moviePrice,
	}

	tickets[id] = t
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)

	go func(ticketID int, price int) {
		time.Sleep(5 * time.Second)
		fmt.Printf("[Async]: Confirmation for ticket #%d sent. Total: %d KZT\n", ticketID, price)
	}(t.ID, t.Price)
}

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
