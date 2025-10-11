#!/bin/bash

set -euo pipefail

cd $(dirname $0)/../

if [[ ${DEBUG:-} != "" ]]; then
    set -x
fi

REGISTRY=${REGISTRY:-'docker.io'}
REPO=${REPO:-'cnrancher'}
TAG=${TAG:-'latest'}

IMAGE_TAG="${REGISTRY}/${REPO}/cilium-egress-operator:${TAG}"
echo "Start build image: $IMAGE_TAG"
set -x

docker build -f package/Dockerfile \
    -t ${IMAGE_TAG} \
    .

set +x
echo "Image: Done"
