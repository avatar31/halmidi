APP_NAME := halmidi
BUILD_DIR := build/bin
GO_BIN := $(shell go env GOPATH)/bin

.PHONY: prereq build test test-all clean

prereq:
	@echo "Checking pre-commit..."
	@command -v pre-commit > /dev/null || { \
		echo "pre-commit is not installed."; \
		echo "Please install it first:"; \
		echo "  brew install pre-commit"; \
		echo "  pipx install pre-commit"; \
		exit 1; \
	}
	@pre-commit install

	@echo "Checking govulncheck..."
	@command -v govulncheck > /dev/null || \
		{ echo "govulncheck is not installed."; \
		  echo "Run: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		  exit 1; }

	@echo "Checking goimports..."
	@command -v goimports > /dev/null || \
		{ echo "goimports is not installed."; \
		  echo "Run: go install golang.org/x/tools/cmd/goimports@latest"; \
		  exit 1; }

	@echo "Checking gofumpt..."
	@command -v gofumpt > /dev/null || \
		{ echo "gofumpt is not installed."; \
		  echo "Run: go install mvdan.cc/gofumpt@latest"; \
		  exit 1; }

	@echo "Checking golangci-lint..."
	@command -v golangci-lint > /dev/null || \
		{ echo "golangci-lint is not installed."; \
		  echo "Check https://golangci-lint.run/docs/welcome/install"; \
		  exit 1; }

	@echo "All prerequisites are available."

precommit:
	@pre-commit run --all-files

build:
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) .

test:
	@go test ./...

test-all:
	go test ./...
	go test -race ./...
	go test -bench=. -benchmem ./...

clean:
	@rm -rf $(BUILD_DIR)
