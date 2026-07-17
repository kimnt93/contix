# contix — build & install
BINARY   := contix
PKG      := .
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X contix/internal/cli.Version=$(VERSION)
DIST     := dist

# --- Host platform detection (for installing the prebuilt binary) ---------
HOST_OS   := $(shell uname -s 2>/dev/null | tr '[:upper:]' '[:lower:]')
HOST_ARCH := $(shell uname -m 2>/dev/null)

ifneq (,$(findstring darwin,$(HOST_OS)))
  OS := darwin
else ifneq (,$(findstring mingw,$(HOST_OS)))
  OS := windows
else ifneq (,$(findstring msys,$(HOST_OS)))
  OS := windows
else ifneq (,$(findstring cygwin,$(HOST_OS)))
  OS := windows
else
  OS := linux
endif

ifneq (,$(filter x86_64 amd64,$(HOST_ARCH)))
  ARCH := amd64
else ifneq (,$(filter aarch64 arm64,$(HOST_ARCH)))
  ARCH := arm64
else
  ARCH := $(HOST_ARCH)
endif

EXT :=
ifeq ($(OS),windows)
  EXT := .exe
endif

PREBUILT := $(DIST)/contix-$(OS)-$(ARCH)$(EXT)

# Install location (no Go required). Override with PREFIX=/usr/local etc.
# Use /usr/local/bin only when it is writable without sudo; otherwise fall
# back to ~/.local/bin.
USR_LOCAL_WRITABLE := $(shell [ -w /usr/local/bin ] && echo yes)
ifeq ($(OS),windows)
  BINDIR ?= $(LOCALAPPDATA)/contix/bin
else ifeq ($(USR_LOCAL_WRITABLE),yes)
  BINDIR ?= /usr/local/bin
else
  BINDIR ?= $(HOME)/.local/bin
endif
ifneq ($(PREFIX),)
  BINDIR := $(PREFIX)/bin
endif

.PHONY: all build install upgrade test vet fmt fmt-check tidy clean release

all: build

## install: install the prebuilt binary for this platform (no build needed)
install:
	@if [ ! -f "$(PREBUILT)" ]; then \
	  echo "error: prebuilt binary not found: $(PREBUILT)"; \
	  echo "       run 'make release' to build it, or check your OS/arch."; \
	  exit 1; \
	fi
	@mkdir -p "$(BINDIR)"
	@cp "$(PREBUILT)" "$(BINDIR)/$(BINARY)$(EXT)"
	@chmod +x "$(BINDIR)/$(BINARY)$(EXT)" 2>/dev/null || true
	@echo "installed $(PREBUILT) -> $(BINDIR)/$(BINARY)$(EXT)"
	@case ":$(PATH):" in *":$(BINDIR):"*) ;; \
	  *) echo "note: $(BINDIR) is not on your PATH; add it to use 'contix'." ;; esac

## upgrade: update this checkout and install the latest committed prebuilt binary
upgrade:
	@git diff --quiet && git diff --cached --quiet || { \
	  echo "error: this checkout has local changes; commit or stash them before upgrading"; \
	  exit 1; \
	}
	git pull --ff-only
	@$(MAKE) install BINDIR="$(BINDIR)"

## build: compile the binary for the host platform into ./$(BINARY) (needs Go)
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

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

## release: cross-compile prebuilt binaries for all supported platforms (needs Go)
release:
	./scripts/build.sh $(VERSION)
