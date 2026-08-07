#!/usr/bin/env bash

set -euo pipefail
cdir="$(readlink -f "$(dirname "$(readlink -f "${0}")")"/..)"

function msg {
    echo "${@}" >&2
}

function die {
    msg "${@}"
    exit 1
}

if [[ $# -eq 0 ]]; then
    msg "usage: ${0} <remote host> [build script options]"
    die "use 'run' as a build script option to run fazantix after installing"
fi

host="${1}"

rsync \
    -rvzza --delete --progress --info=progress2 \
    --exclude="/.git" --exclude="/build" --exclude="/web_ui/node_modules" \
    --filter='P /debian/***' \
    "${cdir}"/ "${host}":/root/fazantix-deploy/

ssh "${host}" "/root/fazantix-deploy/debian/build-and-install.sh ${@:2}"
