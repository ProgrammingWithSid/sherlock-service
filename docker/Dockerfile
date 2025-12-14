# Frontend build stage
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend files
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ .
# Build (type checking removed from build script for Docker)
RUN npm run build

# Go build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app/backend

# Install git for dependencies
RUN apk add --no-cache git

# Set Go environment variables to limit cache usage
ENV GOOS=linux
ENV CGO_ENABLED=0

# Copy go mod files
COPY backend/go.mod ./
COPY backend/go.sum ./

# Copy backend source code
COPY backend/ .

# Generate go.sum and download dependencies
RUN go mod tidy && go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ../server ./cmd/server

# Clean up Go build cache and temp files after build to save space
RUN go clean -cache -testcache || true && \
    rm -rf /tmp/go-build* || true

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy server binary
COPY --from=builder /app/server .

# Copy frontend build
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

EXPOSE 3000

CMD ["./server"]
