#!/bin/bash

BENCH_DIR=".benchmark"
BENCH_CMD=(go test -run='^$' -bench=. -count=6 ./...)

# Ensure the benchmark directory exists
mkdir -p $BENCH_DIR

# Look for an older benchmark run
LAST_BENCH_FILE=$(find "$BENCH_DIR" -maxdepth 1 -name '*.bench' -print0 2>/dev/null | xargs -0 ls -t 2>/dev/null | head -n 1 || true)

# Generate a name for the new file
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
CURRENT_BENCH_FILE="$BENCH_DIR/$TIMESTAMP.bench"

# Running benchmark tests
echo "Running benchmark tests..."
"${BENCH_CMD[@]}" > "$CURRENT_BENCH_FILE"
echo "Done, benchmark results saved to $TIMESTAMP.bench"

if [ -z "$LAST_BENCH_FILE" ]; then
    echo "No previous benchmark file found, Stopping here."
else
    echo "Found older benchmark file: $LAST_BENCH_FILE, running benchstat..."

    # Ensure benchstat is installed
    if ! command -v benchstat &> /dev/null; then
        echo "benchstat could not be found. https://pkg.go.dev/golang.org/x/perf/cmd/benchstat"
        exit 1
    fi
    # Run benchstat to perform the comparison.
    benchstat "$LAST_BENCH_FILE" "$CURRENT_BENCH_FILE"
fi
