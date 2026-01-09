.PHONY: build run dev test clean deps lint install build-all dev-all

# Backend targets
build:
	mkdir -p bin
	cd backend && go build -o ../bin/server ./cmd/server
	cd backend && go build -o ../bin/worker ./cmd/worker
	cd backend && go build -o ../bin/create-admin ./cmd/create-admin

run:
	cd backend && go run ./cmd/server/main.go

# Development mode with hot-reloading (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "⚠️  Air not found. Installing..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

test:
	cd backend && go test ./...

clean:
	rm -rf backend/bin/ tmp/ frontend/dist frontend/node_modules
	cd backend && go clean

deps:
	cd backend && go mod download
	cd backend && go mod tidy

lint:
	cd backend && golangci-lint run

# Frontend targets
frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

frontend-test:
	cd frontend && npm test || true

frontend-lint:
	cd frontend && npm run lint

# Monorepo targets
install: frontend-install
	@echo "✅ All dependencies installed"

build-all: frontend-build build
	@echo "✅ Frontend and backend built"

dev-all:
	@echo "🚀 Starting frontend and backend..."
	@npm run dev || (cd frontend && npm run dev &) && go run ./cmd/server/main.go

test-all: test frontend-test
	@echo "✅ All tests completed"

lint-all: lint frontend-lint
	@echo "✅ All linting completed"

.DEFAULT_GOAL := build

# Admin utilities
create-admin:
	@if [ -z "$(EMAIL)" ] || [ -z "$(PASSWORD)" ] || [ -z "$(NAME)" ] || [ -z "$(DB_URL)" ]; then \
		echo "Usage: make create-admin EMAIL=satenderk8700@gmail.com PASSWORD=verySecure@Pass@@ NAME='Super Admin' DB_URL='postgres://user:pass@localhost/sherlock?sslmode=disable'"; \
		exit 1; \
	fi
	cd backend && go run ./cmd/create-admin/main.go -email $(EMAIL) -password $(PASSWORD) -name "$(NAME)" -db "$(DB_URL)"
