package main

import (
	"log"
	"net/http"
)

func main() {
	store := NewMovieStore()
	h := NewMovieHandler(store)

	http.HandleFunc("/movies", h.Movies)
	http.HandleFunc("/movies/", h.MovieByID)

	log.Println("API running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
