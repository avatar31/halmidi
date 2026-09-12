#!/bin/bash

clear

if ! goreman version > /dev/null 2>&1; then
    go install github.com/mattn/goreman@latest
    echo "Installed goreman $(goreman version)"
fi

echo "Starting Halmidi Dev Cluster ..."
export HALMIDI_S3_ENV="DEV"

# Build the project
cd build || exit
./build_binary.sh
cd ../

# Setup directories as per assets/dev_cluster/*.conf files
for i in {0..2}; do
    mkdir -p "/tmp/halmidi/cluster/node${i}/application"

    for j in {1..6}; do
        mkdir -p "/tmp/halmidi/cluster/node${i}/data/disk${j}"
    done
done

goreman start
