# contix — build & install
BINARY   := contix
PKG      := .
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X contix/internal/cli.Version=$(VERSION)
GOBIN    ?= $(shell go env GOPATH)/bin
DIST     := dist

.PHONY: all build install test vet fmt fmt-check tidy clean release

all: build

## build: compile the binary for the host platform into ./$(BINARY)
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

## install: install contix into $(GOBIN)
install:
	go install -trimpath -ldflags "$(LDFLAGS)" $(PKG)
	@echo "installed to $(GOBIN)/$(BINARY)"

## test: run the test suite
test:
	go test ./...

## vet: run go vet
vet:
	go vet ./...

## fmt: format all Go files
fmt:
	gofmt -w .

## fmt-check: fail if any file is not gofmt-clean
fmt-check:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "unformatted:"; echo "$$out"; exit 1; fi

## tidy: tidy the module
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	rm -rf $(BINARY) $(DIST)

## release: cross-compile release binaries for all supported platforms
release:
	./scripts/build.sh $(VERSION)
