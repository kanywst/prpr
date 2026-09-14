BINARY := prpr
GOLANGCI_LINT_VERSION := v2.13.2

.DEFAULT_GOAL := check

.PHONY: check
check: fmt vet lint test ## Run everything CI runs

.PHONY: build
build: ## Build the binary into ./bin
	go build -trimpath -o bin/$(BINARY) .

.PHONY: install
install: ## Install prpr into $GOBIN
	go install .

.PHONY: run
run: ## Build and run
	go run .

.PHONY: test
test: ## Run tests with the race detector
	go test -race -shuffle=on -coverprofile=coverage.out -covermode=atomic ./...

.PHONY: cover
cover: test ## Show per-function coverage
	go tool cover -func=coverage.out

.PHONY: fmt
fmt: ## Format the tree
	gofmt -l -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run

.PHONY: demo
demo: build ## Re-record docs/demo.gif with VHS
	PATH="$(CURDIR)/bin:$$PATH" vhs docs/demo.tape

.PHONY: tidy
tidy: ## Tidy go.mod/go.sum
	go mod tidy

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin coverage.out dist

.PHONY: help
help: ## List targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
