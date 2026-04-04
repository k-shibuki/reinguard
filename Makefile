# reinguard — optional local dev shortcuts for Go checks only.
# CI remains authoritative; rgd commands should be invoked directly.
# Policy/workflow scripts live under .reinguard/scripts/ — invoke with bash
# (see .github/CONTRIBUTING.md); they are not wrapped here.
# Run `make help` for targets.

.PHONY: help fmt vet test lint build check

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
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not in PATH; see .github/CONTRIBUTING.md or rely on CI" >&2; \
		exit 1; \
	fi
	golangci-lint run --timeout=5m ./...

build: ## Build rgd binary to ./rgd
	go build -o rgd ./cmd/rgd

check: fmt vet lint test ## fmt, vet, golangci-lint, test (local gate)
