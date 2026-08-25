#!/usr/bin/env python3
import glob
import subprocess
import tabulate
import os

import shutil

import importlib.resources
from pathlib import Path

def find_executable(name):
    return shutil.which(name) or shutil.which('./build/' + name) or shutil.which('./' + name)

def bench(cmd):
    p = subprocess.run(cmd, stdout=subprocess.PIPE, text=True, env={"XDG_RUNTIME_DIR": "/tmp"})
    for line in p.stdout.splitlines():
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
    return bench(['xinit', find_executable('fazantix-x11'), '--benchmark', config])

def bench_wayland(config):
    return bench(['cage', '--', find_executable('fazantix-wayland'), '--benchmark', config])

def main():
    result = []
    for config in sorted(Path((importlib.resources.files("fazantbench.config") / "dummy").parent).glob('*.yaml')):
        x11 = bench_x11(config)
        wayland = bench_wayland(config)
        result.append((os.path.basename(config), x11, wayland))
    print()
    print(tabulate.tabulate(result, headers=["config", "X11", "Wayland"]))

if __name__ == "__main__":
    main()
