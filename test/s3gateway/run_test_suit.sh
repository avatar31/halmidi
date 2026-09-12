#!/bin/bash
clear

echo "Setting Test Environment..."
TEST_ROOT_DIR=$(pwd)
mkdir -p "$TEST_ROOT_DIR/bin"

python3 -m venv venv
# trunk-ignore(shellcheck/SC1091)
source venv/bin/activate
pip3 install -r requirements.txt

cd ../../build || exit

# Build the project
./build_binary.sh
mv bin/halmidi "$TEST_ROOT_DIR/bin"

cd "$TEST_ROOT_DIR/bin" || exit

# Create test datapath
mkdir -p halmidi_metadata
mkdir -p data/disk1 data/disk2 data/disk3 data/disk4 data/disk5 data/disk6

export HALMIDI_ENV="DEV"
tmp_config_file=halmidi.conf
sed "s|BASE_PATH|$TEST_ROOT_DIR|g" "$TEST_ROOT_DIR/assets/halmidi.conf" > $tmp_config_file

echo "Starting Halmidi..."
nohup ./halmidi start --dev-config $tmp_config_file > test_suit.log 2>&1 &
BINARY_PID=$!

echo "Halmidi started with PID: $BINARY_PID"

sleep 2

if ! curl -sI http://localhost:9051/api/v1/health > /dev/null; then
    echo "Halmidi Server is not running"
    exit 1
fi

cd ../

echo "Running S3 Gateway Tests..."
pytest -m s3

echo "Tests complete. Stopping Halmidi..."
kill $BINARY_PID

# Wait for graceful exit
sleep 2

if ps -p $BINARY_PID > /dev/null; then
    echo "Force killing process..."
    kill -9 $BINARY_PID
fi

# cleanup
rm -rf "${TEST_ROOT_DIR:?}/bin"
