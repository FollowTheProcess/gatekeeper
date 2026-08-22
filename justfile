_default:
    @just --list

# Tidy dependencies in go.mod
tidy:
    go mod tidy

# Run code generation
gen:
    go generate ./...

# Run the test suite
[env("CGO_ENABLED", "1")]
test:
    go test -race ./...

# Create a local build of the project
build:
    mkdir -p ./bin
    goreleaser build --single-target --skip before --snapshot --clean --output ./bin/gatekeeper
    sam build

# Create a snapshot release
snapshot:
    goreleaser releaser --snapshot --clean

# Run all project benchmarks
bench:
    go test ./... -run None -benchmem -bench .

# Run linting
lint:
    golangci-lint run --fix ./...
    nilaway ./...
    sam validate --lint

# Calculate and show test coverage
cov:
    go test -race -cover -covermode atomic -coverprofile coverage.out ./...
    go tool cover -html coverage.out

# Run testing and linting in one go
check: test lint

# Remove build artifacts and other clutter
clean:
    go clean ./...
    rm -rf coverage.out ./bin ./dist

# Update dependencies in go.mod and go.sum
update:
    go get -u ./...
    go mod tidy
    nix-update --flake default --version skip

# Install the project on your machine
install: uninstall build
    cp ./bin/gatekeeper $GOBIN/gatekeeper

# Uninstall the project from your machine
uninstall:
    rm -rf $GOBIN/gatekeeper
