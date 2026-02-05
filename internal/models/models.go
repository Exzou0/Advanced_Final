package models

import "time"

type Movie struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Genre    string `json:"genre"`
	Duration int    `json:"duration"` // minutes
	Price    int    `json:"price"`
}

type Hall struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	TotalSeats int    `json:"total_seats"`
}

type Seat struct {
	ID     int `json:"id"`
	HallID int `json:"hall_id"`
	Row    int `json:"row"`
	Number int `json:"number"`
}

type Session struct {
	ID      int       `json:"id"`
	MovieID int       `json:"movie_id"`
	HallID  int       `json:"hall_id"`
	Time    time.Time `json:"time"`
	Price   float64   `json:"price"`
}

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type Ticket struct {
	ID        int    `json:"id"`
	SessionID int    `json:"session_id"`
	SeatID    int    `json:"seat_id"`
	UserID    int    `json:"user_id"`
	Status    string `json:"status"` // booked/paid/cancelled
	Price     int    `json:"price"`
}
