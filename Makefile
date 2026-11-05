# Marrow of Empire — Test & Build Makefile
# Usage: make <target>
# Run from project root.
#
# Architecture:
#   src/          — game library (module: marrow-of-empire)
#   kaiju/src/    — Kaiju Engine (module: kaijuengine.com)
#   tests/        — test suites (module: marrow-of-empire-tests)
#
# The engine imports game code via the editor's project system.
# Build the engine from kaiju/src/; the editor loads src/ as the project.

KAiju_DIR  := kaiju/src
CGO       := CGO_ENABLED=1
TAGS      := debug,editor,filedrop,rawsrc
TAGS_AI   := debug,editor,filedrop,rawsrc,ai_driver
GO_TEST   := $(CGO) go test
GO_BUILD  := $(CGO) go build

.PHONY: help test test-unit test-integration test-all \
        bench bench-spatial bench-all \
        build build-editor build-ai-driver clean \
        lint vet run run-ai-driver

# ── Help ───────────────────────────────────────────
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
	  | sort \
	  | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}'

# ── Tests ──────────────────────────────────────────
test: test-unit ## Run unit tests (default)

test-unit: ## Run all unit tests
	$(GO_TEST) ./tests/unit/... -v -count=1

test-integration: ## Run all integration tests
	$(GO_TEST) ./tests/integration/... -v -count=1 -tags="$(TAGS)"

test-all: test-unit test-integration ## Run unit + integration tests

test-quick: ## Run unit tests (no cache bust, fast)
	$(GO_TEST) ./tests/unit/...

# ── Benchmarks ─────────────────────────────────────
bench: ## Run all benchmarks
	$(GO_TEST) ./tests/unit/entity/ -v -count=1 -bench=. -run=^$$

bench-spatial: ## Run only spatial query benchmarks
	$(GO_TEST) ./tests/unit/entity/ -v -count=1 -run TestSpatialQueryPerformance

bench-targets: ## Run all performance target checks (the gate)
	$(GO_TEST) ./tests/unit/entity/ -v -count=1 -run TestPerformanceTargets

# ── Single test ────────────────────────────────────
# Usage: make one TEST=TestCreateEntity_MapsToKaijuEntityId
one: ## Run a single test (pass TEST=<name>)
	$(GO_TEST) ./tests/... -v -count=1 -tags="$(TAGS)" -run $(TEST)

# ── Build ──────────────────────────────────────────
build: ## Build game (editor + game, debug, rawsrc)
	cd $(KAiju_DIR) && $(GO_BUILD) -tags="$(TAGS)" -o ../../bin/game .

build-editor: ## Build editor-only (no rawsrc — uses embedded FS)
	cd $(KAiju_DIR) && $(GO_BUILD) -tags="debug,editor,filedrop" -o ../../bin/editor .

build-ai-driver: ## Build with AI Driver for automated testing/screenshots
	cd $(KAiju_DIR) && $(GO_BUILD) -tags="$(TAGS_AI)" -o ../../bin/game .

build-standalone: ## Build standalone game (no editor — game-only)
	cd $(KAiju_DIR) && $(GO_BUILD) -tags="debug,ai_driver" -o ../../bin/game-standalone .

clean: ## Remove build artifacts
	rm -rf bin/

# ── Run ────────────────────────────────────────────
# The engine must run with CWD=kaiju/ — it resolves editor/ and src/
# paths relative to that directory (rawsrc init: os.OpenRoot("src")).
run: build ## Build and run the game (editor)
	cd kaiju && ../bin/game

run-ai-driver: build-ai-driver ## Build and run with AI Driver (editor)
	cd kaiju && ../bin/game

run-standalone: build-standalone ## Build and run standalone game
	./bin/game-standalone

# ── Code quality ───────────────────────────────────
lint: ## Run go vet on game source
	cd src && go vet ./...

vet: lint ## Alias for lint

lint-engine: ## Run go vet on engine source
	cd $(KAiju_DIR) && go vet ./...
