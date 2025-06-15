.PHONY: bench all

# Run all tests
all: bench

# Run benchmarks
bench:
	@./scripts/benchmark.sh
