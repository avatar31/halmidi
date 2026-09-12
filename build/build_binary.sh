#!/bin/bash
set -e
ARCH="amd64"
APP_NAME="halmidi"
WORKING_DIR=$PWD

rm -rf bin
mkdir -p bin

cd ..
echo "🔨 Building Go binary..."

go mod tidy
GOARCH=$ARCH GOOS=linux go build -buildmode=pie -ldflags="-linkmode=external" -o "$WORKING_DIR/bin/$APP_NAME" main.go
