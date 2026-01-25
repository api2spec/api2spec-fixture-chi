FROM golang:1.22-alpine

WORKDIR /app

# Install build tools
RUN apk add --no-cache gcc musl-dev

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN go build -o server ./main.go

EXPOSE 3000
CMD ["./server"]
