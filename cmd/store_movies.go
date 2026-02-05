package main

import (
	"Final_1/internal/models"
	"errors"
	"strings"
	"sync"
)

type MovieStore struct {
	mu     sync.RWMutex
	nextID int
	items  map[int]models.Movie
}

func NewMovieStore() *MovieStore {
	return &MovieStore{
		nextID: 1,
		items:  make(map[int]models.Movie),
	}
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

	s.mu.Lock()
	defer s.mu.Unlock()

	m.ID = s.nextID
	s.nextID++
	s.items[m.ID] = m
	return m, nil
}

func (s *MovieStore) List() []models.Movie {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]models.Movie, 0, len(s.items))
	for _, m := range s.items {
		out = append(out, m)
	}
	return out
}

func (s *MovieStore) Get(id int) (models.Movie, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.items[id]
	return m, ok
}

type MoviePatch struct {
	Title    *string `json:"title"`
	Genre    *string `json:"genre"`
	Duration *int    `json:"duration"`
	Price    *int    `json:"price"`
}

func (s *MovieStore) Update(id int, p MoviePatch) (models.Movie, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.items[id]
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

	s.items[id] = m
	return m, nil
}

func (s *MovieStore) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return false
	}
	delete(s.items, id)
	return true
}
