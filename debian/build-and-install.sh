#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$(readlink -f "${0}")")"

./generate-release.sh
cd ..
apt-get -y build-dep .
dpkg-buildpackage -b -uc -us

dpkg -i \
    ../fazantix-{wayland,validate-config}{,-dbgsym}_0.0.1_amd64.deb \
    ../fazantix-data_0.0.1_all.deb \
    ../fazantix-frontend_0.0.1_all.deb

if [[ $# -ne 0 ]]; then
    if [[ $1 == "run" ]]; then
        /opt/fazantix/run.sh
    else
        echo "unknown action '${1}'" >&2
        exit 1
    fi
fi
