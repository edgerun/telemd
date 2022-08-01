#!/usr/bin/env bash

image=edgerun/telemd-gpu

if [[ $1 ]]; then
	version="$1"
else
	version=$(git rev-parse --short HEAD)
fi

basetag="${image}:${version}"

# change into project root
BASE="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT=$(realpath "${BASE}/../../")
cd $PROJECT_ROOT

docker run --rm --privileged multiarch/qemu-user-static --reset -p yes

# build all the images
docker build -t ${basetag}-amd64 -f build/package/telemd-gpu/Dockerfile.amd64 .
docker build -t ${basetag}-arm64v8 -f build/package/telemd-gpu/Dockerfile.arm64v8 .

# # push em all
docker push ${basetag}-amd64 &
docker push ${basetag}-arm64v8 &

wait

export DOCKER_CLI_EXPERIMENTAL=enabled

# create the manifest
docker manifest create ${basetag} \
	${basetag}-amd64 \
	${basetag}-arm64v8

# explicit annotations
docker manifest annotate ${basetag} ${basetag}-arm64v8 --os "linux" --arch "arm" --variant "v8"


# ship it
docker manifest push --purge ${basetag}
