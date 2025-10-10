#!/usr/bin/env bash

set -euo pipefail

if ! hash helm 2>/dev/null; then
    exit 0
fi

cd $(dirname $0)/..
WORKINGDIR=$(pwd)

rm -rf build/charts &> /dev/null || true
mkdir -p build dist/artifacts &> /dev/null || true
cp -rf charts build/ &> /dev/null || true

sed -i \
    -e 's/^version:.*/version: '${TAG/v/}'/' \
    -e 's/appVersion:.*/appVersion: '${TAG/v/}'/' \
    build/charts/cilium-egress-operator/Chart.yaml

sed -i \
    -e 's/tag:.*/tag: '${TAG}'/' \
    build/charts/cilium-egress-operator/values.yaml

sed -i \
    -e 's/^version:.*/version: '${TAG/v/}'/' \
    -e 's/appVersion:.*/appVersion: '${TAG/v/}'/' \
    build/charts/cilium-egress-operator-crd/Chart.yaml

helm package -d ./dist/artifacts ./build/charts/cilium-egress-operator
helm package -d ./dist/artifacts ./build/charts/cilium-egress-operator-crd
