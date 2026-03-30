.PHONY: test test-go test-frontend test-e2e test-all lint lint-go lint-frontend build clean coverage-go

# Go test target
test-go:
	go test ./... -count=1

# Frontend unit tests
test-frontend:
	cd frontend && bunx vitest run

# E2E tests (Playwright only)
test-e2e: build
	cd frontend && bunx playwright test

# Run all tests
test: test-go test-frontend

# Run everything including E2E
test-all: test test-e2e

# Lint Go code
lint-go:
	go vet ./...
	golangci-lint run ./...

# Lint frontend
lint-frontend:
	cd frontend && bun run lint
	cd frontend && bun run typecheck

# Lint all
lint: lint-go lint-frontend

# Go coverage report
coverage-go:
	go test ./... -coverprofile=coverage.out -count=1 && go tool cover -func=coverage.out | tail -1

# Build the Go binary
build:
	go build -o ttyweb .

# Clean build artifacts
clean:
	rm -f ttyweb
	rm -rf frontend/dist
