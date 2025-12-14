.PHONY: build run dev test clean deps lint install build-all dev-all deploy deploy-ecr deploy-build setup-ecr-login cleanup-ecr cleanup-server

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

# ECR Configuration
AWS_REGION ?= us-east-1
ECR_REPOSITORY_SERVER ?= sherlock-service/server
ECR_REPOSITORY_WORKER ?= sherlock-service/worker
SSH_KEY ?= ~/Desktop/sherlock-service.pem
SSH_USER ?= ubuntu
SSH_HOST ?= 13.233.117.33

# Deployment helpers
# Admin utilities
create-admin:
	@if [ -z "$(EMAIL)" ] || [ -z "$(PASSWORD)" ] || [ -z "$(NAME)" ] || [ -z "$(DB_URL)" ]; then \
		echo "Usage: make create-admin EMAIL=satenderk8700@gmail.com PASSWORD=verySecure@Pass@@ NAME='Super Admin' DB_URL='postgres://user:pass@localhost/sherlock?sslmode=disable'"; \
		exit 1; \
	fi
	cd backend && go run ./cmd/create-admin/main.go -email $(EMAIL) -password $(PASSWORD) -name "$(NAME)" -db "$(DB_URL)"

# ECR-based deployment (default - uses pre-built images from ECR)
deploy: deploy-ecr

deploy-ecr:
	@echo "🚀 Deploying using ECR images..."
	ssh -i $(SSH_KEY) $(SSH_USER)@$(SSH_HOST) "cd sherlock-service && \
		git fetch origin && git reset --hard origin/main && \
		AWS_REGION=$(AWS_REGION) bash scripts/deploy-ecr.sh"

# Legacy build-on-EC2 deployment (fallback)
deploy-build:
	@echo "🧹 Performing deep cleanup and fresh build..."
	ssh -i $(SSH_KEY) $(SSH_USER)@$(SSH_HOST) "cd sherlock-service && \
		git fetch origin && git reset --hard origin/main && \
		cd docker && \
		docker-compose down && \
		docker system prune -af --volumes && \
		docker builder prune -af && \
		rm -rf /tmp/go-build* && \
		docker-compose -f docker-compose.yml -f docker-compose.build.yml build --no-cache && \
		docker-compose -f docker-compose.yml -f docker-compose.build.yml up -d"

# Setup ECR login on EC2 (one-time setup)
setup-ecr-login:
	@echo "🔐 Setting up ECR login on EC2..."
	ssh -i $(SSH_KEY) $(SSH_USER)@$(SSH_HOST) "cd sherlock-service && bash scripts/ecr-login.sh"

# Manual ECR cleanup trigger
cleanup-ecr:
	@echo "🧹 Triggering ECR cleanup workflow..."
	@echo "   This will clean up old ECR images (keeps last 10)"
	gh workflow run ecr-cleanup.yml || echo "⚠️  GitHub CLI not installed or not authenticated. Run manually from GitHub Actions."

cleanup-server:
	@echo "🧹 Cleaning up Docker resources on server..."
	ssh -i $(SSH_KEY) $(SSH_USER)@$(SSH_HOST) "bash sherlock-service/scripts/ec2-cleanup.sh || (docker system prune -af --volumes && docker builder prune -af && rm -rf /tmp/go-build* && df -h)"

scp:
	scp -i ~/Desktop/sherlock-service.pem \
	  frontend/package.json \
	  docker/Dockerfile \
	  ubuntu@13.233.117.33:~/sherlock-service/
