#!/usr/bin/env bash

set -euo pipefail
cd $(dirname $0)/..
WORKINGDIR=$(pwd)

if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
  echo 'Ensure git directory is clean before run this script:'
  git status --short
  exit 1
fi

echo 'Running: go mod verify'
go mod verify

echo 'Running: go fmt'
go fmt
if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
  echo 'go fmt produced differences'
  exit 1
fi

echo 'Running: go generate'
go generate
if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
  echo 'go generate produced differences'
  exit 1
fi

echo 'Running: go mod tidy'
go mod tidy
if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
  echo 'go mod tidy produced differences'
  exit 1
fi
