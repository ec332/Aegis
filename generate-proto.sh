#!/bin/bash

# Script to generate Go code from proto files

echo "Generating Go code from proto files..."

# Install specific versions for compatibility
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
fi

# Create output directories
mkdir -p proto/gen/market proto/gen/wallet proto/gen/settlement

# Generate Go code for each proto file
for proto_file in proto/*.proto; do
    if [ -f "$proto_file" ]; then
        echo "Processing $proto_file..."
        basename=$(basename "$proto_file" .proto)
        protoc --go_out=proto/gen/$basename --go_opt=paths=source_relative \
               --go-grpc_out=proto/gen/$basename --go-grpc_opt=paths=source_relative \
               -I proto "$proto_file"
    fi
done

echo "Go code generation completed!"