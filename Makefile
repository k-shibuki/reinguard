# reinguard — optional dev shortcuts (not an SSOT; CI remains authoritative).
# No bridle-style evidence targets. Run `make help` for targets.

.PHONY: help fmt vet test lint coverage build check

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

fmt: ## go fmt ./...
	go fmt ./...

vet: ## go vet ./...
	go vet ./...

test: ## go test ./... -race -count=1
	go test ./... -race -count=1

lint: ## golangci-lint (required for check; must be on PATH)
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run --timeout=5m ./... || \
		(echo "golangci-lint not in PATH; install from CONTRIBUTING or rely on CI" >&2; exit 1)

coverage: ## Race tests + module coverage profile + 80% gate
	go test ./... -race -count=1 -coverpkg=./... -coverprofile=coverage.out
	bash tools/check-coverage-threshold.sh 80 coverage.out

build: ## Build rgd binary to ./rgd
	go build -o rgd ./cmd/rgd

check: fmt vet lint test ## fmt, vet, golangci-lint, test (local gate)
