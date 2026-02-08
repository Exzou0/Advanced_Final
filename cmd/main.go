package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort" // Добавили для сортировки кнопок, если нужно
	"strconv"
	"sync"
	"time"

	"Final_1/internal/models"
	_ "github.com/lib/pq" // Драйвер для PostgreSQL
)

var (
	mu sync.Mutex

	// Глобальные переменные для доступа из всех хендлеров
	store *MovieStore
	h     *MovieHandler

	// Мапа остается для совместимости с Milestone 2
	tickets      = map[int]models.Ticket{}
	nextTicketID = 1
)

func main() {
	// 1. Строка подключения
	connStr := "user=postgres password=exzou8520 dbname=ADV host=localhost port=5432 sslmode=disable"

	var err error
	store, err = NewMovieStore(connStr)
	if err != nil {
		log.Fatal("Не удалось подключиться к БД: ", err)
	}

	// Проверка связи
	if err := store.db.Ping(); err != nil {
		log.Fatal("База данных недоступна: ", err)
	}

	// Инициализация Handler
	h = NewMovieHandler(store)
	movieHandler := &MovieHandler{store: store}

	// --- МАРШРУТЫ (ENDPOINTS) ---

	fs := http.FileServer(http.Dir("../web"))
	http.Handle("/", fs)

	// API эндпоинты
	http.HandleFunc("/movies/stats", movieHandler.GetStats)
	http.HandleFunc("/movies/top", movieHandler.GetTopMovies)
	http.HandleFunc("/book", bookHandler)
	http.HandleFunc("/ticket", ticketHandler)
	// Исправлено: передаем store.db
	http.HandleFunc("/tickets", getAllTicketsHandler(store.db))

	// CRUD фильмов
	http.HandleFunc("/movies", h.Movies)
	http.HandleFunc("/movies/", h.MovieByID)

	// 4. Фоновая задача (Health Check)
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

// bookHandler - логика бронирования с записью в БД
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

	movie, ok := store.Get(req.SessionID)
	if !ok {
		http.Error(w, "Фильм не найден", http.StatusNotFound)
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
		Price:     movie.Price,
	}

	// Сохраняем в память
	tickets[id] = t
	mu.Unlock()

	// --- НОВОЕ: Записываем билет в базу данных ---
	// Чтобы getAllTicketsHandler увидел этот билет
	_, err := store.db.Exec(
		"INSERT INTO tickets (id, session_id, seat_id, user_id, price, status) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING",
		t.ID, t.SessionID, t.SeatID, t.UserID, t.Price, t.Status,
	)
	if err != nil {
		log.Printf("[ERROR]: Failed to save ticket to DB: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)

	go func(ticketID int, price int) {
		time.Sleep(5 * time.Second)
		fmt.Printf("[System]: Confirmation for ticket #%d sent. Total: %d KZT\n", ticketID, price)
	}(t.ID, t.Price)
}

func ticketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	mu.Lock()
	t, ok := tickets[id]
	mu.Unlock()

	// Если в памяти нет, попробуем поискать в базе (на случай перезагрузки сервера)
	if !ok {
		err := store.db.QueryRow("SELECT id, session_id, seat_id, user_id, price, status FROM tickets WHERE id = $1", id).
			Scan(&t.ID, &t.SessionID, &t.SeatID, &t.UserID, &t.Price, &t.Status)
		if err != nil {
			http.Error(w, "ticket not found", http.StatusNotFound)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func getAllTicketsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Читаем все ID из базы данных
		rows, err := db.Query("SELECT id FROM tickets ORDER BY id ASC")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type TicketID struct {
			ID int `json:"id"`
		}

		var results []TicketID
		for rows.Next() {
			var t TicketID
			if err := rows.Scan(&t.ID); err == nil {
				results = append(results, t)
			}
		}

		// Если база пуста, вернем хотя бы то, что есть в памяти (для подстраховки)
		if len(results) == 0 {
			mu.Lock()
			for id := range tickets {
				results = append(results, TicketID{ID: id})
			}
			mu.Unlock()
			sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}
