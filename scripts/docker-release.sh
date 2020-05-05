#!/usr/bin/env bash

BASE="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT=$(realpath "${BASE}/../")

if [[ $1 ]]; then
	VERSION="$1"
else
  echo "please specify a version number until we can determine it from the repo"
  exit 1
fi

cd $PROJECT_ROOT

# registry/group/repository/image
IMAGE=git.dsg.tuwien.ac.at:5005/mc2/go-telemetry/telemc

docker build -t ${IMAGE}:${VERSION}-amd64 -f build/package/telemd/Dockerfile.amd64 .
docker build -t ${IMAGE}:${VERSION}-arm32v7 -f build/package/telemd/Dockerfile.arm32v7 .

export DOCKER_CLI_EXPERIMENTAL=enabled

docker push ${IMAGE}:${VERSION}-amd64 &
docker push ${IMAGE}:${VERSION}-arm32v7 &
wait

docker manifest create --amend ${IMAGE}:${VERSION} \
	${IMAGE}:${VERSION}-amd64 \
	${IMAGE}:${VERSION}-arm32v7

docker manifest push ${IMAGE}:${VERSION}
