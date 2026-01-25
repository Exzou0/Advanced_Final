package main

import "time"

type Movie struct {
	ID       int
	Title    string
	Genre    string
	Duration int
}

type Hall struct {
	ID         int
	Name       string
	TotalSeats int
}

type Seat struct {
	ID     int
	HallID int
	Row    int
	Number int
}

type Session struct {
	ID      int
	MovieID int
	HallID  int
	Time    time.Time
	Price   float64
}

type User struct {
	ID    int
	Name  string
	Email string
	Role  string
}

type Ticket struct {
	ID        int
	SessionID int
	SeatID    int
	UserID    int
	Status    string
}
