#!/usr/bin/env python3

import json
import re
import statistics
import sys
from pathlib import Path


def format_rate(rate: float, unit: str) -> str:
    if unit == "bit/s":
        for divisor, suffix in ((1e9, "Gbit/s"), (1e6, "Mbit/s"), (1e3, "kbit/s")):
            if rate >= divisor:
                return f"{rate / divisor:.3f} {suffix}"
    if rate >= 1e6:
        return f"{rate / 1e6:.3f} M {unit}"
    if rate >= 1e3:
        return f"{rate / 1e3:.3f} k {unit}"
    return f"{rate:.3f} {unit}"


def load_results(root: Path):
    results = {}
    raw_root = root / "raw"
    if not raw_root.exists():
        return results
    for path in sorted(raw_root.glob("*/*.json")):
        variant = path.parent.name
        with path.open(encoding="utf-8") as input_file:
            report = json.load(input_file)
        for measurement in report.get("results", []):
            key = (variant, measurement["scenario"])
            results.setdefault(key, []).append(measurement)
    return results


def load_process_metrics(root: Path):
    metrics = {}
    raw_root = root / "raw"
    if not raw_root.exists():
        return metrics
    for proc_path in sorted(raw_root.glob("*/*-process.txt")):
        variant = proc_path.parent.name
        content = proc_path.read_text(encoding="utf-8")
        cpu_ticks_match = re.search(r"cpu_ticks=(\d+)", content)
        vmrss_match = re.search(r"VmRSS:\s+(\d+)\s+kB", content)
        if cpu_ticks_match and vmrss_match:
            entry = metrics.setdefault(variant, {"cpu_ticks": [], "vmrss_mb": [], "idle_ticks": [], "idle_switches": []})
            entry["cpu_ticks"].append(int(cpu_ticks_match.group(1)))
            entry["vmrss_mb"].append(int(vmrss_match.group(1)) / 1024.0)

            idle_path = proc_path.with_name(proc_path.name.replace("-process.txt", "-idle.txt"))
            if idle_path.exists():
                idle_content = idle_path.read_text(encoding="utf-8")
                it_match = re.search(r"idle_ticks=(\d+)", idle_content)
                is_match = re.search(r"idle_switches=(\d+)", idle_content)
                if it_match:
                    entry["idle_ticks"].append(int(it_match.group(1)))
                if is_match:
                    entry["idle_switches"].append(int(is_match.group(1)))
    return metrics


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: summarize.py RESULT_DIRECTORY", file=sys.stderr)
        return 2
    root = Path(sys.argv[1])
    results = load_results(root)

    print("# Transparent inbound benchmark")
    print()
    run_info = root / "environment" / "run.txt"
    if run_info.exists():
        print("```text")
        print(run_info.read_text(encoding="utf-8").strip())
        print("```")
        print()

    if not results:
        print("No valid benchmark reports were produced.")
    else:
        direct = {
            scenario: statistics.median(item["rate"] for item in measurements)
            for (variant, scenario), measurements in results.items()
            if variant == "direct"
        }
        print("| Variant | Scenario | Median | Relative to direct | Runs | Errors |")
        print("|---|---|---:|---:|---:|---:|")
        variant_order = {
            name: index
            for index, name in enumerate(
                (
                    "direct",
                    "ebpf-local",
                    "ebpf-local-tc",
                    "ebpf-shared",
                    "tun-go",
                    "tun-go-auto-redirect",
                    "tun-go-auto-redirect-mq",
                    "tun-mixed",
                    "tun-mixed-auto-redirect",
                )
            )
        }
        scenario_order = {
            name: index
            for index, name in enumerate(
                ("tcp-short", "tcp-upload", "tcp-download", "udp-pps", "udp-unconnected-pps", "udp-churn")
            )
        }
        for (variant, scenario), measurements in sorted(
            results.items(),
            key=lambda item: (variant_order.get(item[0][0], 99), scenario_order.get(item[0][1], 99)),
        ):
            median_rate = statistics.median(item["rate"] for item in measurements)
            baseline = direct.get(scenario)
            relative = "baseline" if variant == "direct" else "N/A"
            if variant != "direct" and baseline:
                relative = f"{median_rate / baseline * 100:.1f}%"
            errors = sum(item.get("errors", 0) for item in measurements)
            print(
                f"| {variant} | {scenario} | {format_rate(median_rate, measurements[0]['unit'])} "
                f"| {relative} | {len(measurements)} | {errors} |"
            )

        proc_metrics = load_process_metrics(root)
        if proc_metrics:
            print()
            print("## Process Resource & Power Metrics")
            print()
            print("| Variant | Active CPU Ticks (Median) | Idle Ticks (3s Median) | Idle Context Switches (3s Median) | Resident Memory VmRSS (Median) |")
            print("|---|---:|---:|---:|---:|")
            for variant in sorted(proc_metrics.keys(), key=lambda v: variant_order.get(v, 99)):
                data = proc_metrics[variant]
                cpu_med = f"{int(statistics.median(data['cpu_ticks'])):,}" if data["cpu_ticks"] else "N/A"
                rss_med = f"{statistics.median(data['vmrss_mb']):.1f} MB" if data["vmrss_mb"] else "N/A"
                idle_t_med = f"{int(statistics.median(data['idle_ticks']))}" if data["idle_ticks"] else "N/A"
                idle_s_med = f"{int(statistics.median(data['idle_switches']))}" if data["idle_switches"] else "N/A"
                print(f"| {variant} | {cpu_med} | {idle_t_med} | {idle_s_med} | {rss_med} |")

    failures = root / "failures.tsv"
    if failures.exists() and failures.read_text(encoding="utf-8").strip():
        print()
        print("## Failures")
        print()
        print("```text")
        print(failures.read_text(encoding="utf-8").strip())
        print("```")

    print()
    print(
        "Hosted-runner results are suitable for functional checks and same-job relative regression only. "
        "Use repeated runs on a fixed self-hosted bare-metal runner for publishable absolute comparisons."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
