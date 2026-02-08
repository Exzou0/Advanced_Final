package main

import (
	"database/sql"
	"errors"
	"strings"

	"Final_1/internal/models"
	_ "github.com/lib/pq"
)

type MovieStore struct {
	db *sql.DB
}
type MoviePatch struct {
	Title    *string  `json:"title"`
	Genre    *string  `json:"genre"`
	Duration *int     `json:"duration"`
	Price    *int     `json:"price"`
	Rating   *float64 `json:"rating"`
}

func NewMovieStore(connStr string) (*MovieStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &MovieStore{db: db}, nil
}

func (s *MovieStore) Create(m models.Movie) (models.Movie, error) {
	m.Title = strings.TrimSpace(m.Title)
	m.Genre = strings.TrimSpace(m.Genre)

	if m.Title == "" {
		return models.Movie{}, errors.New("title is required")
	}
	if m.Duration <= 0 {
		return models.Movie{}, errors.New("duration must be > 0")
	}
	if m.Price < 0 {
		return models.Movie{}, errors.New("price cannot be negative")
	}
	if m.Rating < 0 || m.Rating > 5 {
		return models.Movie{}, errors.New("rating must be between 0 and 5")
	}

	query := `INSERT INTO movies (title, genre, duration, price, rating) 
          VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err := s.db.QueryRow(query, m.Title, m.Genre, m.Duration, m.Price, m.Rating).Scan(&m.ID)
	if err != nil {
		return models.Movie{}, err
	}

	return m, nil
}

func (s *MovieStore) GetAll() ([]models.Movie, error) {
	query := `SELECT id, title, genre, duration, price, rating FROM movies ORDER BY id ASC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []models.Movie
	for rows.Next() {
		var m models.Movie
		if err := rows.Scan(&m.ID, &m.Title, &m.Genre, &m.Duration, &m.Price, &m.Rating); err != nil {
			return nil, err
		}
		movies = append(movies, m)
	}
	return movies, nil
}

func (s *MovieStore) Get(id int) (models.Movie, bool) {
	var m models.Movie
	query := `SELECT id, title, genre, duration, price, rating FROM movies WHERE id = $1`

	err := s.db.QueryRow(query, id).Scan(&m.ID, &m.Title, &m.Genre, &m.Duration, &m.Price, &m.Rating)
	if err != nil {
		return m, false
	}
	return m, true
}

func (s *MovieStore) Update(id int, p MoviePatch) (models.Movie, error) {
	m, ok := s.Get(id)
	if !ok {
		return models.Movie{}, errors.New("movie not found")
	}

	if p.Title != nil {
		m.Title = strings.TrimSpace(*p.Title)
		if m.Title == "" {
			return models.Movie{}, errors.New("title cannot be empty")
		}
	}
	if p.Genre != nil {
		m.Genre = strings.TrimSpace(*p.Genre)
	}
	if p.Duration != nil {
		if *p.Duration <= 0 {
			return models.Movie{}, errors.New("duration must be > 0")
		}
		m.Duration = *p.Duration
	}
	if p.Price != nil {
		if *p.Price < 0 {
			return models.Movie{}, errors.New("price cannot be negative")
		}
		m.Price = *p.Price
	}
	if p.Rating != nil {
		if *p.Rating < 0 || *p.Rating > 5 {
			return models.Movie{}, errors.New("rating must be between 0 and 10")
		}
		m.Rating = *p.Rating
	}

	query := `UPDATE movies SET title=$1, genre=$2, duration=$3, price=$4, rating=$5 WHERE id=$6`
	_, err := s.db.Exec(query, m.Title, m.Genre, m.Duration, m.Price, m.Rating, id)
	if err != nil {
		return models.Movie{}, err
	}

	return m, nil
}

func (s *MovieStore) Delete(id int) bool {
	query := `DELETE FROM movies WHERE id = $1`
	result, err := s.db.Exec(query, id)
	if err != nil {
		return false
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0
}
func (s *MovieStore) GetTopRated() ([]models.Movie, error) {
	query := `SELECT id, title, genre, duration, price, rating FROM movies ORDER BY rating DESC, title ASC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []models.Movie
	for rows.Next() {
		var m models.Movie
		if err := rows.Scan(&m.ID, &m.Title, &m.Genre, &m.Duration, &m.Price, &m.Rating); err != nil {
			return nil, err
		}
		movies = append(movies, m)
	}
	return movies, nil
}

type GenreStat struct {
	Genre string `json:"genre"`
	Count int    `json:"count"`
}

type YearlyStats struct {
	TotalMinutes int            `json:"total_minutes"`
	TotalMovies  int            `json:"total_movies"`
	TopGenres    []GenreStat    `json:"top_genres"` // Изменили map на слайс []GenreStat
	TopMovies    []models.Movie `json:"top_movies"`
}

func (s *MovieStore) GetYearlyStats() (YearlyStats, error) {
	var stats YearlyStats

	stats.TopGenres = []GenreStat{}

	queryMain := `
        SELECT 
            COALESCE(SUM(m.duration), 0), 
            COUNT(t.id) 
        FROM movies m 
        JOIN tickets t ON m.id = t.session_id`

	err := s.db.QueryRow(queryMain).Scan(&stats.TotalMinutes, &stats.TotalMovies)
	if err != nil {
		return stats, err
	}

	queryGenres := `
        SELECT m.genre, COUNT(t.id) as sales 
        FROM movies m 
        JOIN tickets t ON m.id = t.session_id 
        GROUP BY m.genre 
        ORDER BY sales DESC 
        LIMIT 3`

	rows, err := s.db.Query(queryGenres)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var gs GenreStat
			// Сканируем жанр и количество продаж в структуру gs
			if err := rows.Scan(&gs.Genre, &gs.Count); err == nil {
				stats.TopGenres = append(stats.TopGenres, gs)
			}
		}
	}

	return stats, nil
}
