package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type HealthStatus struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Post struct {
	ID     int    `json:"id"`
	UserID int    `json:"userId"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Health routes
	r.Get("/health", healthHandler)
	r.Get("/health/ready", readyHandler)

	// User routes
	r.Route("/users", func(r chi.Router) {
		r.Get("/", listUsers)
		r.Post("/", createUser)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getUser)
			r.Put("/", updateUser)
			r.Delete("/", deleteUser)
			r.Get("/posts", getUserPosts)
		})
	})

	// Post routes
	r.Route("/posts", func(r chi.Router) {
		r.Get("/", listPosts)
		r.Post("/", createPost)
		r.Get("/{id}", getPost)
	})

	http.ListenAndServe(":8080", r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(HealthStatus{Status: "ok", Version: "0.1.0"})
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(HealthStatus{Status: "ready", Version: "0.1.0"})
}

func listUsers(w http.ResponseWriter, r *http.Request) {
	users := []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
	}
	json.NewEncoder(w).Encode(users)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	json.NewEncoder(w).Encode(User{ID: id, Name: "Sample User", Email: "user@example.com"})
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	json.NewDecoder(r.Body).Decode(&user)
	user.ID = 1
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var user User
	json.NewDecoder(r.Body).Decode(&user)
	user.ID = id
	json.NewEncoder(w).Encode(user)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func getUserPosts(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(chi.URLParam(r, "id"))
	posts := []Post{{ID: 1, UserID: userID, Title: "User Post", Body: "Content"}}
	json.NewEncoder(w).Encode(posts)
}

func listPosts(w http.ResponseWriter, r *http.Request) {
	posts := []Post{
		{ID: 1, UserID: 1, Title: "First Post", Body: "Hello world"},
		{ID: 2, UserID: 1, Title: "Second Post", Body: "Another post"},
	}
	json.NewEncoder(w).Encode(posts)
}

func getPost(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	json.NewEncoder(w).Encode(Post{ID: id, UserID: 1, Title: "Sample Post", Body: "Post body"})
}

func createPost(w http.ResponseWriter, r *http.Request) {
	var post Post
	json.NewDecoder(r.Body).Decode(&post)
	post.ID = 1
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}
