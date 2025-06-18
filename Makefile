.PHONY: bench gen

# Run benchmarks
bench:
	@./scripts/benchmark.sh

# Code generation
gen:
	@go generate ./...
