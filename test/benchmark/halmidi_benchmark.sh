#!/bin/bash

cd ../..
pwd

go test -bench=. -benchmem ./...

# for pkg in $(go list ./...); do
#     go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof $pkg
#     echo ""
#     echo "$pkg"

#     # Analyse profiles
#     go tool pprof -top cpu.prof
#     go tool pprof -top mem.prof
#     echo ""
# done
