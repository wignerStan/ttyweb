set shell := ["bash", "-cu"]

# Default: list available recipes
default:
    @just --list --list-heading $'Available recipes:\n'

# ── Build ──────────────────────────────────────────────

# Build the Go binary
[group("build")]
build:
    go build -o ttyweb .

# ── Testing ────────────────────────────────────────────

# Run Go unit tests
[group("test")]
test-go:
    go test ./... -count=1

# Run frontend unit tests (vitest)
[group("test")]
test-frontend:
    cd frontend && bunx vitest run

# Run Go and frontend tests
[group("test")]
test: test-go test-frontend

# Run E2E tests (Playwright, requires built binary)
[group("test")]
test-e2e: build
    cd frontend && bunx playwright test

# Run all tests including E2E
[group("test")]
test-all: test test-e2e

# ── Linting ────────────────────────────────────────────

# Lint Go code (vet + golangci-lint)
[group("lint")]
lint-go:
    go vet ./...
    golangci-lint run ./...

# Lint frontend (biome + tsc)
[group("lint")]
lint-frontend:
    cd frontend && bun run lint
    cd frontend && bun run typecheck

# Lint all (Go + frontend)
[group("lint")]
lint: lint-go lint-frontend

# ── Coverage ───────────────────────────────────────────

# Print Go test coverage percentage
[group("test")]
coverage-go:
    go test ./... -coverprofile=coverage.out -count=1 && grep -v "main.go:" coverage.out | grep -v "ws_speech.go:" > coverage_filtered.out && go tool cover -func=coverage_filtered.out | tail -1 && rm -f coverage_filtered.out

# ── Cleanup ────────────────────────────────────────────

# Remove build artifacts (binary + coverage + frontend dist)
[group("build")]
[confirm]
clean:
    rm -f ttyweb
    rm -f coverage.out
    rm -rf frontend/dist
