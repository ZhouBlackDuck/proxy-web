.PHONY: dev-frontend dev-backend build build-frontend build-backend docker-build docker-up docker-down docker-logs clean release

# Version info (injected at build time)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.BuildTime=$(BUILD_TIME) \
	-X main.Commit=$(COMMIT)

# Development
dev-frontend:
	cd frontend && npm run dev

dev-backend:
	cd backend && DATA_DIR=./data go run ./cmd/server/

dev:
	@echo "Starting frontend and backend in parallel..."
	@make dev-frontend & make dev-backend & wait

# Build
build: build-frontend build-backend

build-frontend:
	cd frontend && npm run build

build-backend:
	cd backend && CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o server ./cmd/server/

# Cross-compilation
build-linux-amd64:
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o server-linux-amd64 ./cmd/server/

build-linux-arm64:
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o server-linux-arm64 ./cmd/server/

build-all: build-linux-amd64 build-linux-arm64

# Docker
docker-build:
	docker build -t proxy-web:$(VERSION) -t proxy-web:latest .

docker-build-no-cache:
	docker build --no-cache -t proxy-web:$(VERSION) -t proxy-web:latest .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-rebuild:
	docker compose down
	docker build -t proxy-web:latest .
	docker compose up -d

# Clean
clean:
	rm -rf frontend/dist backend/server backend/server-linux-*
	rm -rf data/

# Test
test-backend:
	cd backend && go test ./...

test-frontend:
	cd frontend && npm run build

# Install dependencies
install-frontend:
	cd frontend && npm install

install-backend:
	cd backend && go mod download

install: install-frontend install-backend

# Release (tag push triggers GitHub Actions workflow)
release:
	@TAG="$${VERSION:-v$$(date +%Y.%m.%d)}"; \
	echo "Creating tag $$TAG and pushing ..."; \
	git tag $$TAG && git push origin $$TAG
