#!/bin/bash

set -euo pipefail

cd "$(dirname "$(readlink -f "${0}")")"

go run 'github.com/fosdem/vidmix/cmd/mixer' 
