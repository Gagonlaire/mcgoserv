.PHONY: all bench gen fetch build clean

BINARY_NAME=server
LDFLAGS_PKG=github.com/Gagonlaire/mcgoserv/internal/buildinfo
BUILD_TIME=$(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
BRANCH=$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)
LDFLAGS=-X '$(LDFLAGS_PKG).BuildTime=$(BUILD_TIME)' -X '$(LDFLAGS_PKG).Stable=true' -X '$(LDFLAGS_PKG).Branch=$(BRANCH)'

all: build

bench:
	@./scripts/benchmark.sh

fetch:
	@./scripts/fetch.sh

gen: fetch
	@go generate ./...

build: gen
	@go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)

clean:
	@rm -f $(BINARY_NAME) server-*.jar
	@rm -rf versions logs libraries internal/mcdata

field-alignment:
	@fieldalignment ./...