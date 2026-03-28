.PHONY: test test-go test-frontend test-e2e test-all lint lint-go lint-frontend build clean

# Go test target
test-go:
	go test ./... -v -count=1

# Frontend unit tests
test-frontend:
	cd frontend && npx vitest run

# E2E tests (Playwright only)
test-e2e: build
	cd frontend && npx playwright test

# Run all tests
test: test-go test-frontend

# Run everything including E2E
test-all: test test-e2e

# Lint Go code
lint-go:
	go vet ./...

# Lint frontend
lint-frontend:
	cd frontend && npx tsc --noEmit

# Lint all
lint: lint-go lint-frontend

# Build the Go binary
build:
	go build -o ttyweb .

# Clean build artifacts
clean:
	rm -f ttyweb
	rm -rf frontend/dist
