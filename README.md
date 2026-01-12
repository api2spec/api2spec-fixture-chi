# api2spec-fixture-chi

A Chi (Go) API fixture for testing api2spec framework detection and route extraction.

## Prerequisites

- Go 1.21 or later

## Installation

```bash
go mod tidy
```

## Build

```bash
go build -o api2spec-fixture-chi .
```

## Run

```bash
./api2spec-fixture-chi
```

The server will start on port 8080.

## API Endpoints

### Health

- `GET /health` - Health check
- `GET /health/ready` - Readiness check

### Users

- `GET /users` - List all users
- `POST /users` - Create a new user
- `GET /users/{id}` - Get a user by ID
- `PUT /users/{id}` - Update a user by ID
- `DELETE /users/{id}` - Delete a user by ID
- `GET /users/{id}/posts` - Get posts for a user

### Posts

- `GET /posts` - List all posts
- `POST /posts` - Create a new post
- `GET /posts/{id}` - Get a post by ID
