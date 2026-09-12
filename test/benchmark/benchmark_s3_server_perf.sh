#!/bin/bash

# HOST = 10.1.63.232:9000
# ACCESS_KEY = LZQIV7UP69UXLCC8L3TTC4BN2YO30MCP
# SCRET_KEY = WDN0QAH3US1OYRECPR55VBER03S4TH56BIGOXTWU3XLSP4Z

if ! command -v warp 2>&1 >/dev/null
then
    echo "warp command not found. Please check here for installation https://github.com/minio/warp"
    exit 1
fi

export S3_SERVER_HOST="10.18.104.157"
export S3_ACCESS_KEY="AFAQDA2AI7D5B55YUAQ"
export S3_SECRET_KEY="s5IqiD5vfn+hQlgNne199fDBH+sCQ0zKEYABsBHMdVo="

if [[ -z "${S3_SERVER_HOST}" ]];
then
    echo "S3_SERVER_HOST not found. Please set the S3 Host address in S3_SERVER_HOST environment variable"
    exit 1
fi

if [[ -z "${S3_ACCESS_KEY}" ]];
then
    echo "S3_ACCESS_KEY not found. Please set the S3 access-key in S3_ACCESS_KEY environment variable"
    exit 1
fi

if [[ -z "${S3_SECRET_KEY}" ]];
then
    echo "S3_SECRET_KEY not found. Please set the S3 secret-key in S3_SECRET_KEY environment variable"
    exit 1
fi

num_jobs=20
benchmark_type="mixed"

if [[ "$benchmark_type" == "mixed" ]]; then
    benchmark_type_with_args="${benchmark_type} --stat-distrib=0"
elif [[ "$benchmark_type" == "get" ]] || [[ "$benchmark_type" == "put" ]]; then
    benchmark_type_with_args="${benchmark_type}"
else
    echo "Invalid benchmark type. Exiting test suit..."
    exit 1
fi

no_of_trials=1
trial=0

while [ $trial -lt $no_of_trials ]; do
    trial=$((trial+1))

    declare -a arr=("64K" "128K" "256K" "512K" "1M" "2M" "4M" "8M")
    # declare -a arr=("64K")
    output_file="output.txt"

    for i in "${arr[@]}"
    do
        echo "-------------------------------------------------------------------------------------------------------------------------------------------------"
        echo "Starting IO test with object size: ${i}"
        echo " "

        tmp_file="output_${i}.txt"

        cmd="warp ${benchmark_type_with_args} --host=${S3_SERVER_HOST} --access-key=${S3_ACCESS_KEY} --secret-key=${S3_SECRET_KEY} --concurrent=${num_jobs} --duration 120s --obj.size=${i}"
        
        echo "Running command:"
        echo "$cmd"

        eval "$cmd"

        if [ $? -ne 0 ]; then
            echo " "
            echo "Error running warp command. Exiting test suit..."
            exit 1
        fi

        warp analyze warp-$benchmark_type-*.json.zst > $tmp_file
        python3 warp_output_analyzer.py -analyze $i $tmp_file

        rm $tmp_file
        rm warp-$benchmark_type-*.json.zst

        echo " "
        echo "-------------------------------------------------------------------------------------------------------------------------------------------------"
    done

    # Aggregate results
    python3 warp_output_analyzer.py -print benchmark_performance.json
done
