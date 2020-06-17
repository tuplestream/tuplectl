#!/usr/bin/env bash

set -euf -o pipefail

export GOARCH=amd64
export CGO_ENABLED=0

if [[ " $@ " =~ " -release" ]]; then
  mkdir -p bin
  BUILD_DATE="$(date -u)"
  VERSION=$(cat VERSION)
  COMMIT=$(git rev-parse --short HEAD)

  GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Commit=$COMMIT -X main.Version=$VERSION -X 'main.BuildDate=$BUILD_DATE'" -a -o bin/tuplectl-linux-amd64 .
  GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Commit=$COMMIT -X main.Version=$VERSION -X 'main.BuildDate=$BUILD_DATE'" -a -o bin/tuplectl-darwin-amd64 .
else
  go build -a -o tuplectl .
fi
