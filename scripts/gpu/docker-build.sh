#!/usr/bin/env bash

# script to build x86 docker images for local usage.
# we're assuming that you are using a an x86 machine.

BASE="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT=$(realpath "${BASE}/../../")
cd $PROJECT_ROOT
docker build -t edgerun/telemd-gpu:$(git rev-parse --short HEAD)-amd64 -f build/package/telemd-gpu/Dockerfile.amd64 .
