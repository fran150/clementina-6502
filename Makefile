# Project variables
BINARY_NAME=clementina
MAIN_PACKAGE=./cmd/clementina.go
GO=go

.PHONY: all build build-all release clean test coverage lint vet fmt bench profile check-docs godoc-serve

# Build variables
BUILD_DIR=build/bin
TEST_DIR=tests
VERSION?=1.0.0
COMMIT=$(shell git rev-parse --short HEAD)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}"

# Test variables
COVERAGE_DIR=${TEST_DIR}/coverage
PROFILES_DIR=${TEST_DIR}/profiles
BENCH_FILE=${PROFILES_DIR}/clementina6502.prof
BENCH_PACKAGE=github.com/fran150/clementina-6502/tests
BENCH_TESTS=^BenchmarkProcessor


.PHONY: all build clean test coverage lint vet fmt bench profile help build-all release

all: clean build test ## Default target: clean, build and test

build: ## Build the binary for current platform
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ${MAIN_PACKAGE}

build-all: clean ## Build for all platforms (Linux, macOS, Windows)
	@echo "Building for all platforms..."
	@mkdir -p ${BUILD_DIR}
	
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 ${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-v${VERSION}-linux-amd64 ${MAIN_PACKAGE}
	GOOS=linux GOARCH=arm64 ${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-v${VERSION}-linux-arm64 ${MAIN_PACKAGE}
	GOOS=linux GOARCH=386 ${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-v${VERSION}-linux-386 ${MAIN_PACKAGE}
	
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 ${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-v${VERSION}-darwin-amd64 ${MAIN_PACKAGE}
	GOOS=darwin GOARCH=arm64 ${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-v${VERSION}-darwin-arm64 ${MAIN_PACKAGE}
	
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 ${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-v${VERSION}-windows-amd64.exe ${MAIN_PACKAGE}
	GOOS=windows GOARCH=386 ${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-v${VERSION}-windows-386.exe ${MAIN_PACKAGE}
	
	@echo "Build complete! Binaries are available in ${BUILD_DIR}"

release: build-all ## Create release packages for all platforms
	@echo "Creating release packages..."
	
	@# Create directories for assets
	@mkdir -p ${BUILD_DIR}/assets/images
	
	@# Linux packages
	@for arch in amd64 arm64 386; do \
		echo "Creating package for linux-$$arch..."; \
		package_dir="${BUILD_DIR}/clementina-linux-$$arch"; \
		mkdir -p "$$package_dir/assets/computer/beneater"; \
		mkdir -p "$$package_dir/assets/images"; \
		cp "${BUILD_DIR}/${BINARY_NAME}-v${VERSION}-linux-$$arch" "$$package_dir/${BINARY_NAME}"; \
		chmod +x "$$package_dir/${BINARY_NAME}"; \
		cp ./scripts/setup-linux.sh "$$package_dir/"; \
		chmod +x "$$package_dir/setup-linux.sh"; \
		cp ./assets/computer/beneater/*.bin "$$package_dir/assets/computer/beneater/" 2>/dev/null || true; \
		cp ./assets/images/computer.jpeg "$$package_dir/assets/images/" 2>/dev/null || true; \
		cp ./README.md "$$package_dir/"; \
		cd "${BUILD_DIR}" && zip -r "${BINARY_NAME}-v${VERSION}-linux-$$arch.zip" "clementina-linux-$$arch" && cd - > /dev/null; \
		rm -rf "$$package_dir"; \
	done
	
	@# macOS packages
	@for arch in amd64 arm64; do \
		echo "Creating package for darwin-$$arch..."; \
		package_dir="${BUILD_DIR}/clementina-darwin-$$arch"; \
		mkdir -p "$$package_dir/assets/computer/beneater"; \
		mkdir -p "$$package_dir/assets/images"; \
		cp "${BUILD_DIR}/${BINARY_NAME}-v${VERSION}-darwin-$$arch" "$$package_dir/${BINARY_NAME}"; \
		chmod +x "$$package_dir/${BINARY_NAME}"; \
		cp ./scripts/setup-macos.sh "$$package_dir/"; \
		chmod +x "$$package_dir/setup-macos.sh"; \
		cp ./assets/computer/beneater/*.bin "$$package_dir/assets/computer/beneater/" 2>/dev/null || true; \
		cp ./assets/images/computer.jpeg "$$package_dir/assets/images/" 2>/dev/null || true; \
		cp ./README.md "$$package_dir/"; \
		cd "${BUILD_DIR}" && zip -r "${BINARY_NAME}-v${VERSION}-darwin-$$arch.zip" "clementina-darwin-$$arch" && cd - > /dev/null; \
		rm -rf "$$package_dir"; \
	done
	
	@# Windows packages
	@for arch in amd64 386; do \
		echo "Creating package for windows-$$arch..."; \
		package_dir="${BUILD_DIR}/${BINARY_NAME}-v${VERSION}-windows-$$arch"; \
		mkdir -p "$$package_dir/assets/computer/beneater"; \
		mkdir -p "$$package_dir/assets/images"; \
		cp "${BUILD_DIR}/${BINARY_NAME}-v${VERSION}-windows-$$arch.exe" "$$package_dir/${BINARY_NAME}.exe"; \
		cp ./assets/computer/beneater/*.bin "$$package_dir/assets/computer/beneater/" 2>/dev/null || true; \
		cp ./assets/images/computer.jpeg "$$package_dir/assets/images/" 2>/dev/null || true; \
		cp ./README.md "$$package_dir/"; \
		cd "${BUILD_DIR}" && zip -r "${BINARY_NAME}-v${VERSION}-windows-$$arch.zip" "${BINARY_NAME}-v${VERSION}-windows-$$arch" && cd - > /dev/null; \
	done
	
	@echo "Release packages are available in ${BUILD_DIR}"

clean: ## Clean build directory
	@echo "Cleaning..."
	@rm -rf ${BUILD_DIR}
	@rm -rf ${COVERAGE_DIR}
	@rm -rf ${PROFILES_DIR}

test: ## Run tests
	@echo "Running tests..."
	${GO} test ./...

# Add a target for verbose testing
test-verbose:
	@echo "Running tests in verbose mode..."
	go test -v ./...


coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@mkdir -p ${COVERAGE_DIR}
	${GO} test -coverprofile=tests/coverage/coverage.txt -covermode=atomic ./...
	${GO} tool cover -html=coverage.txt -o ${COVERAGE_DIR}/coverage.html

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint is not installed. Please install it first."; \
		exit 1; \
	fi

check-docs: ## Check for undocumented exported symbols
	@echo "Checking documentation..."
	@./scripts/check-docs.sh

godoc-serve: ## Start a local godoc server
	@echo "Starting godoc server at http://localhost:6060/pkg/github.com/fran150/clementina-6502/"
	@if command -v godoc >/dev/null; then \
		godoc -http=:6060; \
	else \
		echo "godoc is not installed. Installing..."; \
		go install golang.org/x/tools/cmd/godoc@latest; \
		godoc -http=:6060; \
	fi
vet: ## Run go vet
	@echo "Running go vet..."
	${GO} vet ./...

fmt: ## Run go fmt
	@echo "Running go fmt..."
	${GO} fmt ./...

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@mkdir -p ${PROFILES_DIR}
	${GO} test -benchmem -run=^$$ -bench ${BENCH_TESTS}$$ ${BENCH_PACKAGE} -cpuprofile ${BENCH_FILE}

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
