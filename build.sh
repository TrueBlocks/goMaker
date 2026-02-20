#!/bin/bash

# Create bin directory
mkdir -p bin

# Get current timestamp
COMPILE_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')

# Build with compile time injection
go build -ldflags "-X 'main.compileTime=$COMPILE_TIME'" -o bin/goMaker main.go

echo "Build complete. Compile time set to: $COMPILE_TIME"