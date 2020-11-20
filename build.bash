#!/usr/bin/env bash

set -euf -o pipefail

export GOARCH=amd64
export CGO_ENABLED=0

BUILD_DATE="$(date -u)"
VERSION=$(cat VERSION)
COMMIT=$(git rev-parse --short HEAD)

if [[ " $@ " =~ " -release" ]]; then
  mkdir -p bin

  # go vet
  GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Commit=$COMMIT -X main.Version=$VERSION -X 'main.BuildDate=$BUILD_DATE' -X main.DefaultClientID=$AUTH0_CLIENT_ID -X main.DefaultTenantURL=$AUTH0_TENANT_URL" -a -o bin/tuplectl-linux-amd64 .
  GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Commit=$COMMIT -X main.Version=$VERSION -X 'main.BuildDate=$BUILD_DATE' -X main.DefaultClientID=$AUTH0_CLIENT_ID -X main.DefaultTenantURL=$AUTH0_TENANT_URL" -a -o bin/tuplectl-darwin-amd64 .
else
  VERSION="$VERSION-$(hostname -f)-local"
  COMMIT="$COMMIT-$(git rev-parse --abbrev-ref HEAD)"
  go build -ldflags "-X main.Commit=$COMMIT -X 'main.Version=$VERSION' -X 'main.BuildDate=$BUILD_DATE'  -X main.DefaultClientID=$AUTH0_CLIENT_ID -X main.DefaultTenantURL=$AUTH0_TENANT_URL" -a -o tuplectl .
fi
