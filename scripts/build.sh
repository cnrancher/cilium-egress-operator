#!/bin/bash

cd $(dirname $0)/../

set -euo pipefail

if [[ ${DEBUG:-} != "" ]]; then
    set -x
fi

TAG=${TAG:-'v0.0.0'}
COMMIT=${COMMIT:-'head'}

mkdir -p build

echo "Start build cilium-egress-operator $TAG - $COMMIT"

CGO_ENABLED=0 go build \
    -buildmode=pie \
    -ldflags="-extldflags='-static' -s -w -X github.com/cnrancher/cilium-egress-operator/pkg/utils.Version=${TAG} -X github.com/cnrancher/cilium-egress-operator/pkg/utils.Commit=${COMMIT}" \
    -o build/cilium-egress-operator \
    .

echo '-----------------------------------------------------------------------'
ls -alh ./build/cilium-egress-operator
echo '-----------------------------------------------------------------------'

echo "build: Done"
