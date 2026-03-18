#!/usr/bin/env bash

if ! command -v golangci-lint &> /dev/null; then
    echo "golangci-lint not found."
    exit 1
fi

if ! command -v govulncheck &> /dev/null; then
    echo "govulncheck not found."
    exit 1
fi

if ! command -v nilaway &> /dev/null; then
    echo "nilaway not found."
    exit 1
fi

echo "Running golangci-lint..."
golangci-lint run --output.sarif.path golangci.sarif
echo "golangci-lint completed."

echo "Running govulncheck..."
govulncheck -format sarif ./... > govulncheck.sarif
echo "govulncheck completed."

echo "Running nilaway..."
nilaway -json ./... > nilaway.json
echo "nilaway completed."
