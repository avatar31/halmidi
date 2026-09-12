#!/bin/bash

CURRENT_DIR=$(pwd)

clear
cd ../../
pwd

pkgs=$(go list ./... | grep -v test | grep -v config)
go test -v -short -p 1 $pkgs -coverprofile=${CURRENT_DIR}/unit_coverage.out
echo ""
go tool cover -func ${CURRENT_DIR}/unit_coverage.out
go tool cover -html ${CURRENT_DIR}/unit_coverage.out -o ${CURRENT_DIR}/index.html

echo ""
