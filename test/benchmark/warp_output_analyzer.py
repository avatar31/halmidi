import re
import sys
import json


def analyze_output_file(filename):
    with open(filename, "r") as file:
        content = file.read()
    
    perf = {}
    
    get_match = re.search(r'Report: GET.*?Average: ([\d.]+) MiB/s, ([\d.]+) obj/s', content, re.DOTALL)
    if get_match:
        get_throughput_mib = float(get_match.group(1))
        get_obj_rate = float(get_match.group(2))
        # Convert MiB/s to bytes/s (1 MiB = 1024 * 1024 bytes)
        get_bw_bytes = int(get_throughput_mib * 1024 * 1024)
        perf["GET"] = {
            "bwInBytes": get_bw_bytes,
            "objPerSec": get_obj_rate
        }
    
    put_match = re.search(r'Report: PUT.*?Average: ([\d.]+) MiB/s, ([\d.]+) obj/s', content, re.DOTALL)
    if put_match:
        put_throughput_mib = float(put_match.group(1))
        put_obj_rate = float(put_match.group(2))
        # Convert MiB/s to bytes/s
        put_bw_bytes = int(put_throughput_mib * 1024 * 1024)
        perf["PUT"] = {
            "bwInBytes": put_bw_bytes,
            "objPerSec": put_obj_rate
        }
    
    total_match = re.search(r'Report: Total.*?Average: ([\d.]+) MiB/s, ([\d.]+) obj/s', content, re.DOTALL)
    if total_match:
        total_throughput_mib = float(total_match.group(1))
        total_obj_rate = float(total_match.group(2))
        # Convert MiB/s to bytes/s
        total_bw_bytes = int(total_throughput_mib * 1024 * 1024)
        perf["TOTAL"] = {
            "bwInBytes": total_bw_bytes,
            "objPerSec": total_obj_rate
        }

    return perf


def update_json_performance(object_size, output_file):
    json_file = "benchmark_performance.json"

    performance = analyze_output_file(output_file)
    try:
        with open(json_file, "r") as file:
            data = json.load(file)
    except FileNotFoundError:
        data = {}
    
    data[object_size] = performance
    
    with open(json_file, "w") as file:
        json.dump(data, file, indent=4)


def print_json_in_tabular_format(json_file):
    try:
        with open(json_file, "r") as file:
            data = json.load(file)
    except FileNotFoundError:
        print("No performance data found.")
        return
    print("\nCommand:\nwarp mixed --stat-distrib=0 --host=10.18.104.157 --access-key=<ACCESS_KEY>\n --secret-key=<SECRET_KEY> --concurrent=20 --duration 120s\n --obj.size={object_size}\n")
    print("="*65)
    print(f"{'Object Size':<11} | {'GET BW (MiB/s)':<14} | {'PUT BW (MiB/s)':<14} | {'Total BW (MiB/s)':<14}")
    print("="*65)

    for obj_size, ops in data.items():
        get_bw = ops.get("GET", {}).get("bwInBytes", 0)
        put_bw = ops.get("PUT", {}).get("bwInBytes", 0)
        total_bw = ops.get("TOTAL", {}).get("bwInBytes", 0)

        get_bw_mib = round(get_bw / 1024 / 1024, 2) if get_bw > 0 else 0
        put_bw_mib = round(put_bw / 1024 / 1024, 2) if put_bw > 0 else 0
        total_bw_mib = round(total_bw / 1024 / 1024, 2) if total_bw > 0 else 0

        print(f"{obj_size:<11} | {get_bw_mib:<14} | {put_bw_mib:<14} | {total_bw_mib:<14}")
    print("="*65)


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 warp_output_analyzer.py [-analyze|-print] [options]")
        sys.exit(1)
    
    if sys.argv[1] == "-analyze":
        if len(sys.argv) != 4:
            print("Usage: python3 warp_output_analyzer.py -analyze <object_size> <output_file>")
            sys.exit(1)
        object_size = sys.argv[2]
        output_file = sys.argv[3]
        update_json_performance(object_size, output_file)
    elif sys.argv[1] == "-print":
        if len(sys.argv) != 3:
            print("Usage: python3 warp_output_analyzer.py -print <file_name>")
            sys.exit(1)
        filename = sys.argv[2]
        print_json_in_tabular_format(filename)
