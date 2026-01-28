package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRouter creates a new chi router with all routes configured for testing.
func setupRouter() *chi.Mux {
	r := chi.NewRouter()

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

	return r
}

// ========== Health Endpoint Tests ==========

func assertJSONContentType(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	contentType := w.Header().Get("Content-Type")
	assert.Contains(t, contentType, "application/json")
}

func TestHealthHandler_Success(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assertJSONContentType(t, w)

	var response HealthStatus
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, "0.1.0", response.Version)
}

func TestReadyHandler_Success(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response HealthStatus
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ready", response.Status)
	assert.Equal(t, "0.1.0", response.Version)
}

// ========== User Endpoint Tests ==========

func TestListUsers_Success(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assertJSONContentType(t, w)

	var users []User
	err := json.Unmarshal(w.Body.Bytes(), &users)
	require.NoError(t, err)

	assert.Len(t, users, 2)
	names := []string{users[0].Name, users[1].Name}
	assert.ElementsMatch(t, []string{"Alice", "Bob"}, names)
}

func TestGetUser_Success(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assertJSONContentType(t, w)

	var user User
	err := json.Unmarshal(w.Body.Bytes(), &user)
	require.NoError(t, err)

	assert.Equal(t, 42, user.ID)
	assert.Equal(t, "Sample User", user.Name)
	assert.Equal(t, "user@example.com", user.Email)
}

func TestGetUser_DifferentIDs(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		expectedID int
	}{
		{
			name:       "user id 1",
			userID:     "1",
			expectedID: 1,
		},
		{
			name:       "user id 100",
			userID:     "100",
			expectedID: 100,
		},
		{
			name:       "user id 999",
			userID:     "999",
			expectedID: 999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter()

			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.userID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var user User
			err := json.Unmarshal(w.Body.Bytes(), &user)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedID, user.ID)
		})
	}
}

func TestCreateUser_Success(t *testing.T) {
	router := setupRouter()

	newUser := User{
		Name:  "Charlie",
		Email: "charlie@example.com",
	}
	body, err := json.Marshal(newUser)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assertJSONContentType(t, w)

	var createdUser User
	err = json.Unmarshal(w.Body.Bytes(), &createdUser)
	require.NoError(t, err)

	assert.Equal(t, 1, createdUser.ID)
	assert.Equal(t, "Charlie", createdUser.Name)
	assert.Equal(t, "charlie@example.com", createdUser.Email)
}

func TestCreateUser_EmptyBody(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Handler accepts any valid JSON and assigns ID=1
	assert.Equal(t, http.StatusCreated, w.Code)

	var createdUser User
	err := json.Unmarshal(w.Body.Bytes(), &createdUser)
	require.NoError(t, err)
	assert.Equal(t, 1, createdUser.ID)
}

func TestUpdateUser_Success(t *testing.T) {
	router := setupRouter()

	updatedUser := User{
		Name:  "Alice Updated",
		Email: "alice.updated@example.com",
	}
	body, err := json.Marshal(updatedUser)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var user User
	err = json.Unmarshal(w.Body.Bytes(), &user)
	require.NoError(t, err)

	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "Alice Updated", user.Name)
	assert.Equal(t, "alice.updated@example.com", user.Email)
}

func TestUpdateUser_DifferentIDs(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		expectedID int
	}{
		{
			name:       "update user 5",
			userID:     "5",
			expectedID: 5,
		},
		{
			name:       "update user 123",
			userID:     "123",
			expectedID: 123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter()

			updatedUser := User{
				Name:  "Updated Name",
				Email: "updated@example.com",
			}
			body, err := json.Marshal(updatedUser)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPut, "/users/"+tt.userID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var user User
			err = json.Unmarshal(w.Body.Bytes(), &user)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedID, user.ID)
		})
	}
}

func TestDeleteUser_Success(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodDelete, "/users/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.Bytes())
}

func TestGetUserPosts_Success(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/users/1/posts", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var posts []Post
	err := json.Unmarshal(w.Body.Bytes(), &posts)
	require.NoError(t, err)

	assert.Len(t, posts, 1)
	assert.Equal(t, 1, posts[0].UserID)
	assert.Equal(t, "User Post", posts[0].Title)
}

func TestGetUserPosts_DifferentUserIDs(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		expectedUserID int
	}{
		{
			name:           "user 1 posts",
			userID:         "1",
			expectedUserID: 1,
		},
		{
			name:           "user 42 posts",
			userID:         "42",
			expectedUserID: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter()

			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.userID+"/posts", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var posts []Post
			err := json.Unmarshal(w.Body.Bytes(), &posts)
			require.NoError(t, err)

			assert.Len(t, posts, 1)
			assert.Equal(t, tt.expectedUserID, posts[0].UserID)
		})
	}
}

// ========== Post Endpoint Tests ==========

func TestListPosts_Success(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assertJSONContentType(t, w)

	var posts []Post
	err := json.Unmarshal(w.Body.Bytes(), &posts)
	require.NoError(t, err)

	assert.Len(t, posts, 2)
	titles := []string{posts[0].Title, posts[1].Title}
	assert.ElementsMatch(t, []string{"First Post", "Second Post"}, titles)
}

func TestGetPost_Success(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assertJSONContentType(t, w)

	var post Post
	err := json.Unmarshal(w.Body.Bytes(), &post)
	require.NoError(t, err)

	assert.Equal(t, 1, post.ID)
	assert.Equal(t, "Sample Post", post.Title)
	assert.Equal(t, "Post body", post.Body)
}

func TestGetPost_DifferentIDs(t *testing.T) {
	tests := []struct {
		name       string
		postID     string
		expectedID int
	}{
		{
			name:       "post id 1",
			postID:     "1",
			expectedID: 1,
		},
		{
			name:       "post id 50",
			postID:     "50",
			expectedID: 50,
		},
		{
			name:       "post id 999",
			postID:     "999",
			expectedID: 999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter()

			req := httptest.NewRequest(http.MethodGet, "/posts/"+tt.postID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var post Post
			err := json.Unmarshal(w.Body.Bytes(), &post)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedID, post.ID)
		})
	}
}

func TestCreatePost_Success(t *testing.T) {
	router := setupRouter()

	newPost := Post{
		UserID: 1,
		Title:  "My New Post",
		Body:   "This is the content of my new post",
	}
	body, err := json.Marshal(newPost)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assertJSONContentType(t, w)

	var createdPost Post
	err = json.Unmarshal(w.Body.Bytes(), &createdPost)
	require.NoError(t, err)

	assert.Equal(t, 1, createdPost.ID)
	assert.Equal(t, 1, createdPost.UserID)
	assert.Equal(t, "My New Post", createdPost.Title)
	assert.Equal(t, "This is the content of my new post", createdPost.Body)
}

func TestCreatePost_EmptyBody(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Handler accepts any valid JSON and assigns ID=1
	assert.Equal(t, http.StatusCreated, w.Code)

	var createdPost Post
	err := json.Unmarshal(w.Body.Bytes(), &createdPost)
	require.NoError(t, err)
	assert.Equal(t, 1, createdPost.ID)
}

// ========== Error Cases ==========

func TestNotFound_InvalidRoute(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMethodNotAllowed_WrongMethod(t *testing.T) {
	router := setupRouter()

	// PATCH is not defined for /users
	req := httptest.NewRequest(http.MethodPatch, "/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Chi returns 405 Method Not Allowed for unhandled methods on existing routes
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestCreateUser_InvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// The handler validates the body and returns 400 for invalid JSON
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestCreatePost_InvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// The handler validates the body and returns 400 for invalid JSON
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestGetUser_InvalidPathParam(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Non-numeric ID returns 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetPost_InvalidPathParam(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/posts/abc", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Non-numeric ID returns 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ========== Table-Driven Tests for Comprehensive Coverage ==========

func TestAllEndpoints_StatusCodes(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
	}{
		// Health endpoints
		{"GET /health", http.MethodGet, "/health", nil, http.StatusOK},
		{"GET /health/ready", http.MethodGet, "/health/ready", nil, http.StatusOK},

		// User endpoints
		{"GET /users", http.MethodGet, "/users", nil, http.StatusOK},
		{"POST /users", http.MethodPost, "/users", User{Name: "Test", Email: "test@example.com"}, http.StatusCreated},
		{"GET /users/1", http.MethodGet, "/users/1", nil, http.StatusOK},
		{"PUT /users/1", http.MethodPut, "/users/1", User{Name: "Updated", Email: "updated@example.com"}, http.StatusOK},
		{"DELETE /users/1", http.MethodDelete, "/users/1", nil, http.StatusNoContent},
		{"GET /users/1/posts", http.MethodGet, "/users/1/posts", nil, http.StatusOK},

		// Post endpoints
		{"GET /posts", http.MethodGet, "/posts", nil, http.StatusOK},
		{"POST /posts", http.MethodPost, "/posts", Post{UserID: 1, Title: "Test", Body: "Content"}, http.StatusCreated},
		{"GET /posts/1", http.MethodGet, "/posts/1", nil, http.StatusOK},

		// 404 cases
		{"GET /nonexistent", http.MethodGet, "/nonexistent", nil, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter()

			var reqBody *bytes.Reader
			if tt.body != nil {
				body, err := json.Marshal(tt.body)
				require.NoError(t, err)
				reqBody = bytes.NewReader(body)
			} else {
				reqBody = bytes.NewReader(nil)
			}

			req := httptest.NewRequest(tt.method, tt.path, reqBody)
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "unexpected status code for %s %s", tt.method, tt.path)
		})
	}
}

// ========== Concurrent Request Tests ==========
// These tests verify thread-safety of the handlers under concurrent access.

func TestConcurrentReads_Users(t *testing.T) {
	router := setupRouter()
	const numRequests = 100

	var wg sync.WaitGroup
	wg.Add(numRequests)

	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/users", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("expected status 200, got %d", w.Code)
				return
			}

			var users []User
			if err := json.Unmarshal(w.Body.Bytes(), &users); err != nil {
				errors <- fmt.Errorf("failed to unmarshal response: %w", err)
				return
			}

			if len(users) != 2 {
				errors <- fmt.Errorf("expected 2 users, got %d", len(users))
				return
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestConcurrentReads_Posts(t *testing.T) {
	router := setupRouter()
	const numRequests = 100

	var wg sync.WaitGroup
	wg.Add(numRequests)

	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/posts", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("expected status 200, got %d", w.Code)
				return
			}

			var posts []Post
			if err := json.Unmarshal(w.Body.Bytes(), &posts); err != nil {
				errors <- fmt.Errorf("failed to unmarshal response: %w", err)
				return
			}

			if len(posts) != 2 {
				errors <- fmt.Errorf("expected 2 posts, got %d", len(posts))
				return
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestConcurrentCreates_Users(t *testing.T) {
	router := setupRouter()
	const numRequests = 50

	var wg sync.WaitGroup
	wg.Add(numRequests)

	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			defer wg.Done()

			user := User{
				Name:  fmt.Sprintf("User%d", idx),
				Email: fmt.Sprintf("user%d@example.com", idx),
			}
			body, err := json.Marshal(user)
			if err != nil {
				errors <- fmt.Errorf("failed to marshal user: %w", err)
				return
			}

			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				errors <- fmt.Errorf("expected status 201, got %d", w.Code)
				return
			}

			var created User
			if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
				errors <- fmt.Errorf("failed to unmarshal response: %w", err)
				return
			}

			// Verify the response has an ID assigned
			if created.ID == 0 {
				errors <- fmt.Errorf("expected non-zero ID, got 0")
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestConcurrentCreates_Posts(t *testing.T) {
	router := setupRouter()
	const numRequests = 50

	var wg sync.WaitGroup
	wg.Add(numRequests)

	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			defer wg.Done()

			post := Post{
				UserID: 1,
				Title:  fmt.Sprintf("Post%d", idx),
				Body:   fmt.Sprintf("Content for post %d", idx),
			}
			body, err := json.Marshal(post)
			if err != nil {
				errors <- fmt.Errorf("failed to marshal post: %w", err)
				return
			}

			req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				errors <- fmt.Errorf("expected status 201, got %d", w.Code)
				return
			}

			var created Post
			if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
				errors <- fmt.Errorf("failed to unmarshal response: %w", err)
				return
			}

			// Verify the response has an ID assigned
			if created.ID == 0 {
				errors <- fmt.Errorf("expected non-zero ID, got 0")
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestConcurrentUpdates_Users(t *testing.T) {
	router := setupRouter()
	const numRequests = 50

	var wg sync.WaitGroup
	wg.Add(numRequests)

	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			defer wg.Done()

			user := User{
				Name:  fmt.Sprintf("UpdatedUser%d", idx),
				Email: fmt.Sprintf("updated%d@example.com", idx),
			}
			body, err := json.Marshal(user)
			if err != nil {
				errors <- fmt.Errorf("failed to marshal user: %w", err)
				return
			}

			// All goroutines update the same user ID to stress test
			req := httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("expected status 200, got %d", w.Code)
				return
			}

			var updated User
			if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
				errors <- fmt.Errorf("failed to unmarshal response: %w", err)
				return
			}

			// Verify the ID in response matches the requested ID
			if updated.ID != 1 {
				errors <- fmt.Errorf("expected ID 1, got %d", updated.ID)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestConcurrentMixedOperations(t *testing.T) {
	router := setupRouter()
	const numOpsPerType = 30

	var wg sync.WaitGroup
	// 4 types of operations: list users, list posts, create user, create post
	totalOps := numOpsPerType * 4
	wg.Add(totalOps)

	errors := make(chan error, totalOps)

	// Concurrent list users
	for i := 0; i < numOpsPerType; i++ {
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/users", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("list users: expected 200, got %d", w.Code)
			}
		}()
	}

	// Concurrent list posts
	for i := 0; i < numOpsPerType; i++ {
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/posts", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("list posts: expected 200, got %d", w.Code)
			}
		}()
	}

	// Concurrent create users
	for i := 0; i < numOpsPerType; i++ {
		go func(idx int) {
			defer wg.Done()

			user := User{Name: fmt.Sprintf("MixedUser%d", idx), Email: fmt.Sprintf("mixed%d@example.com", idx)}
			body, _ := json.Marshal(user)

			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				errors <- fmt.Errorf("create user: expected 201, got %d", w.Code)
			}
		}(i)
	}

	// Concurrent create posts
	for i := 0; i < numOpsPerType; i++ {
		go func(idx int) {
			defer wg.Done()

			post := Post{UserID: 1, Title: fmt.Sprintf("MixedPost%d", idx), Body: "Content"}
			body, _ := json.Marshal(post)

			req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				errors <- fmt.Errorf("create post: expected 201, got %d", w.Code)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}
