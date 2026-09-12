#!/bin/bash

# This script generates the protobuf code for the project.

# Check if protoc is installed
if ! command -v protoc &> /dev/null
then
    echo "protoc could not be found. Please install it and try again."
    exit
fi

# Generate the protobuf code
cd internal/core/models || exit
protoc --go_out=. models.proto
protoc --descriptor_set_out=models.pb --include_imports --include_source_info models.proto
