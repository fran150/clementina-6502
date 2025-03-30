# Project variables
BINARY_NAME=clementina
MAIN_PACKAGE=./cmd/clementina.go
GO=go

# Build variables
BUILD_DIR=build
VERSION?=1.0.0
COMMIT=$(shell git rev-parse --short HEAD)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}"

# Test variables
COVERAGE_DIR=coverage
BENCH_FILE=clementina6502.prof

.PHONY: all build clean test coverage lint vet fmt bench profile help

all: clean build test ## Default target: clean, build and test

build: ## Build the binary
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ${MAIN_PACKAGE}

clean: ## Clean build directory
	@echo "Cleaning..."
	@rm -rf ${BUILD_DIR}
	@rm -rf ${COVERAGE_DIR}
	@rm -f ${BENCH_FILE}

test: ## Run tests
	@echo "Running tests..."
	${GO} test -v ./...

coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@mkdir -p ${COVERAGE_DIR}
	${GO} test -coverprofile=${COVERAGE_DIR}/coverage.out ./...
	${GO} tool cover -html=${COVERAGE_DIR}/coverage.out -o ${COVERAGE_DIR}/coverage.html

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint is not installed. Please install it first."; \
		exit 1; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	${GO} vet ./...

fmt: ## Run go fmt
	@echo "Running go fmt..."
	${GO} fmt ./...

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	${GO} test -benchmem -run=^$$ -bench ^BenchmarkProcessor$$ github.com/fran150/clementina6502/tests -cpuprofile ${BENCH_FILE}

profile: bench ## Run profiler UI (requires bench first)
	@echo "Starting profiler UI..."
	${GO} tool pprof -http :8080 ${BENCH_FILE}

run: build ## Run the application
	@echo "Running ${BINARY_NAME}..."
	./${BUILD_DIR}/${BINARY_NAME}

install-deps: ## Install development dependencies
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

help: ## Display this help message
	@echo "Usage:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
