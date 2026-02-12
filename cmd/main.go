package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"Final_1/internal/models"
	_ "github.com/lib/pq"
)

var (
	mu sync.Mutex

	store *MovieStore
	h     *MovieHandler

	tickets = map[int]models.Ticket{}
)

func main() {
	connStr := "user=postgres password=exzou8520 dbname=ADV host=localhost port=5432 sslmode=disable"

	var err error
	store, err = NewMovieStore(connStr)
	if err != nil {
		log.Fatal("Unable to connect to the database: ", err)
	}

	if err := store.db.Ping(); err != nil {
		log.Fatal("Database is unavailable: ", err)
	}

	h = NewMovieHandler(store)
	movieHandler := &MovieHandler{store: store}

	fs := http.FileServer(http.Dir("../web"))
	http.Handle("/", fs)

	adminOnly := AuthMiddleware("admin")
	anyUser := AuthMiddleware("user")

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/movies", h.Movies)
	http.HandleFunc("/movies/top", movieHandler.GetTopMovies)

	http.HandleFunc("/book", anyUser(bookHandler))
	http.HandleFunc("/ticket", anyUser(ticketHandler))
	http.HandleFunc("/tickets", anyUser(getAllTicketsHandler(store.db)))
	http.HandleFunc("/movies/stats", anyUser(movieHandler.GetStats))

	http.HandleFunc("/register", registerHandler)

	http.HandleFunc("/movies/", adminOnly(h.MovieByID))

	go func() {
		for {
			time.Sleep(20 * time.Second)
			mu.Lock()
			count := len(tickets)
			mu.Unlock()
			fmt.Printf("[SYSTEM]: Health check OK. Tickets in memory: %d. DB connected.\n", count)
		}
	}()

	fmt.Println("Server running at: http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func bookHandler(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(userEmailKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, _, err := store.GetUserByEmail(email)
	if err != nil {
		log.Printf("[ERROR]: User not found: %v", err)
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	var req struct {
		SessionID int `json:"session_id"`
		SeatID    int `json:"seat_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	movie, ok := store.Get(req.SessionID)
	if !ok {
		http.Error(w, "Movie not found", http.StatusNotFound)
		return
	}

	var newID int
	query := `
		INSERT INTO tickets (session_id, seat_id, user_id, price, status) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id`

	err = store.db.QueryRow(query, req.SessionID, req.SeatID, user.ID, movie.Price, "BOOKED").Scan(&newID)
	if err != nil {
		log.Printf("[ERROR]: Failed to save ticket: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	t := models.Ticket{
		ID:        newID,
		SessionID: req.SessionID,
		SeatID:    req.SeatID,
		UserID:    user.ID,
		Status:    "BOOKED",
		Price:     movie.Price,
	}

	mu.Lock()
	tickets[newID] = t
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)

	go func(ticketID int, price int) {
		time.Sleep(3 * time.Second)
		fmt.Printf("[SYSTEM]: Ticket #%d confirmed for %s. Price: %d KZT\n", ticketID, email, price)
	}(t.ID, t.Price)
}

func ticketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid ticket id", http.StatusBadRequest)
		return
	}

	email, ok := r.Context().Value(userEmailKey).(string)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, _, err := store.GetUserByEmail(email)
	if err != nil {
		http.Error(w, "user not found", http.StatusForbidden)
		return
	}

	var t models.Ticket
	query := `SELECT id, session_id, seat_id, user_id, price, COALESCE(status, 'BOOKED') FROM tickets WHERE id = $1`
	err = store.db.QueryRow(query, id).Scan(&t.ID, &t.SessionID, &t.SeatID, &t.UserID, &t.Price, &t.Status)

	if err != nil {
		log.Printf("[DEBUG]: Scan error for ticket %d: %v", id, err)
		http.Error(w, "ticket error", http.StatusInternalServerError)
		return
	}

	fmt.Printf("DEBUG: TicketOwnerID=%d, CurrentUserID=%d\n", t.UserID, user.ID)
	if user.Role != "admin" && t.UserID != user.ID {
		log.Printf("[SECURITY]: User %s tried to access ticket %d owned by UserID %d", email, id, t.UserID)
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func getAllTicketsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email, ok := r.Context().Value(userEmailKey).(string)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			return
		}

		user, _, err := store.GetUserByEmail(email)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "User profile sync error"})
			return
		}

		var rows *sql.Rows
		query := "SELECT id, price, status FROM tickets"

		if user.Role == "admin" {
			rows, err = db.Query(query + " ORDER BY id DESC")
		} else {
			rows, err = db.Query(query+" WHERE user_id = $1 ORDER BY id DESC", user.ID)
		}

		if err != nil {
			http.Error(w, "Database query error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type TicketItem struct {
			ID     int    `json:"id"`
			Price  int    `json:"price"`
			Status string `json:"status"`
		}

		var results []TicketItem
		for rows.Next() {
			var t TicketItem
			if err := rows.Scan(&t.ID, &t.Price, &t.Status); err == nil {
				results = append(results, t)
			}
		}

		if results == nil {
			results = []TicketItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

var jwtKey = []byte("your_secret_key_2026")

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&credentials)

	user, hash, err := store.GetUserByEmail(credentials.Email)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(credentials.Password))
	if err != nil {
		http.Error(w, "Wrong password", http.StatusUnauthorized)
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &struct {
		Email string `json:"email"`
		Role  string `json:"role"`
		jwt.RegisteredClaims
	}{
		Email:            user.Email,
		Role:             user.Role,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expirationTime)},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(jwtKey)

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  expirationTime,
		Path:     "/",
		HttpOnly: true,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"role": user.Role,
		"name": user.Name,
		"id":   user.ID,
	})
}

type contextKey string

const userEmailKey contextKey = "userEmail"

func AuthMiddleware(requiredRole string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			claims := &struct {
				Email string `json:"email"`
				Role  string `json:"role"`
				jwt.RegisteredClaims
			}{}

			token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
				return jwtKey, nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if requiredRole == "admin" && claims.Role != "admin" {
				http.Error(w, "Forbidden: Admins only", http.StatusForbidden)
				return
			}
			if requiredRole == "user" && (claims.Role != "user" && claims.Role != "admin") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userEmailKey, claims.Email)
			next(w, r.WithContext(ctx))
		}
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	role := "user"

	if strings.HasSuffix(strings.ToLower(data.Email), "@admin.com") {
		role = "admin"
		log.Printf("[SYSTEM]: New admin detected: %s", data.Email)
	}

	err := store.CreateUser(data.Name, data.Email, data.Password, role)
	if err != nil {
		log.Printf("[REGISTER ERROR]: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User created successfully",
		"role":    role,
	})
}
