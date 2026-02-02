.PHONY: all bench gen fetch build clean

BINARY_NAME=server

all: build

bench:
	@./scripts/benchmark.sh

fetch:
	@./scripts/fetch.sh

gen: fetch
	@go generate ./...

build: gen
	@go build -o $(BINARY_NAME)

clean:
	@rm -f $(BINARY_NAME) server-*.jar
	@rm -rf versions logs libraries internal/mcdata