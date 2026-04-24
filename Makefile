BIN       := dist/dis
BIN_LINUX := dist/dis-linux-amd64

.PHONY: all build build-linux test test-integration release redo-release clean

all: build

## build: compile dis for the current platform
build:
	go build -o $(BIN) .

## build-linux: compile dis for Linux/amd64 (used by integration tests)
build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BIN_LINUX) .

## test: run unit tests
test:
	go test ./...

## test-integration: build the Linux binary then run integration tests in Docker
test-integration: build-linux
	DISGO_BIN=$(shell pwd)/$(BIN_LINUX) \
		go test -v -tags integration -run TestInstallIntegration -timeout 120s ./tests/

## release: tag and push a new release (triggers goreleaser via GitHub Actions)
## Usage: make release VERSION=v0.1.0
release:
	@if [ -z "$(VERSION)" ]; then echo "Usage: make release VERSION=v0.x.x"; exit 1; fi
	git tag $(VERSION)
	git push origin $(VERSION)

redo-release:
	@if [ -z "$(VERSION)" ]; then echo "Usage: make release VERSION=v0.x.x"; exit 1; fi
	git tag -d $(VERSION) && git push origin :$(VERSION)
	git tag $(VERSION)
	git push origin $(VERSION)

## clean: remove build artifacts
clean:
	rm -rf dist/
