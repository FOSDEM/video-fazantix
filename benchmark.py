#!/usr/bin/env python3
import glob
import subprocess
import tabulate

def bench(cmd):
    p = subprocess.run(cmd, stdout=subprocess.PIPE, text=True)
    for line in p.splitlines():
        if not line.startswith("BENCHMARK:"):
            continue
        part = line.split()
        data = {}
        for p in part:
            if ':' in p:
                key, val = p.split(":", maxsplit=1)
                data[key] = val
        return data["avg"]

def bench_x11(config):
    return bench(['xinit', './build/fazantix-x11', '--benchmark', config])

def bench_wayland(config):
    return bench(['cage', '--', './build/fazantix-wayland', '--benchmark', config])

def main():
    result = []
    for config in glob.glob("examples/benchmark/*.yaml"):
        x11 = bench_x11(config)
        wayland = bench_wayland(config)
        result.append((os.path.basename(config), x11, wayland))
    print()
    print(tabulate(result, headers=["config", "X11", "Wayland"]))

if __name__ == "__main__":
    main()