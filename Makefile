BINARY    := admin-stats
BUILD_DIR := ./bin
SEED_CMD  := ./cmd/seed

.PHONY: help \
        run seed \
        build clean \
        test test-race test-verbose test-integration test-all test-cover \
        fmt fmt-check vet lint tidy \
        check pre-pr \
        install-tools

# ── Help ──────────────────────────────────────────────────────────────────────

help: ## Show available targets
	@echo ""
	@echo "  Development"
	@grep -E '^(run|seed):.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "    \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "  Build"
	@grep -E '^(build|clean):.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "    \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "  Testing"
	@grep -E '^test[a-z-]*:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "    \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "  Code Quality"
	@grep -E '^(fmt|fmt-check|vet|lint|tidy):.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "    \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "  Workflow"
	@grep -E '^(check|pre-pr|install-tools):.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "    \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""

# ── Development ───────────────────────────────────────────────────────────────

run: ## Start the server (reads conf/default.toml)
	go run main.go

seed: ## Seed MongoDB with 2M rounds / 4M transactions
	go run $(SEED_CMD)/main.go

# ── Build ─────────────────────────────────────────────────────────────────────

build: ## Compile binary into ./bin/admin-stats
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) .

clean: ## Remove ./bin/
	@rm -rf $(BUILD_DIR)

# ── Testing ───────────────────────────────────────────────────────────────────

test: ## Unit tests — no infrastructure needed (fast)
	go test ./...

test-race: ## Unit tests with race detector
	go test -race ./...

test-verbose: ## Unit tests with full output (-v)
	go test -v ./...

test-integration: ## Repository integration tests — requires running MongoDB
	go test -tags integration -timeout 60s -count=1 -v ./repository/...

test-all: ## All tests including integration — requires running MongoDB
	go test -tags integration -timeout 60s -count=1 ./...

test-cover: ## Unit tests + open HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report → coverage.html"

# ── Code Quality ──────────────────────────────────────────────────────────────

fmt: ## Format all Go files in-place
	gofmt -w .

fmt-check: ## Check formatting without writing (exits non-zero if dirty)
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "\033[31mUnformatted files (run 'make fmt' to fix):\033[0m"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	@echo "\033[32mFormatting OK\033[0m"

vet: ## Run go vet across all packages
	go vet ./...

lint: ## Run staticcheck (install via: make install-tools)
	staticcheck ./...

tidy: ## Tidy go.mod and verify the module graph
	go mod tidy
	go mod verify

# ── Workflow ──────────────────────────────────────────────────────────────────

# Fast local check — no MongoDB needed. Run this during development to catch
# issues before they reach CI.
check: fmt-check vet lint test ## Quick check: fmt + vet + lint + unit tests
	@echo ""
	@echo "\033[32m✓ All checks passed\033[0m"

# Full pre-PR gate — identical to what CI should run. Requires MongoDB for
# integration tests. Run this before opening a pull request.
pre-pr: fmt-check vet lint test-all ## Full pre-PR gate: fmt + vet + lint + all tests (needs MongoDB)
	@echo ""
	@echo "\033[32m✓ Ready to raise a PR\033[0m"

# ── Tools ─────────────────────────────────────────────────────────────────────

install-tools: ## Install staticcheck
	go install honnef.co/go/tools/cmd/staticcheck@latest
