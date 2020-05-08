#!/usr/bin/env bash

export GOARCH=amd64
export CGO_ENABLED=0

if [[ " $@ " =~ " -release" ]]; then
  mkdir -p bin
  sed -i "s/AUTOREPLACED-VERSION/$(cat VERSION)/" main.go
  GOOS=linux go get github.com/zalando/go-keyring
  GOOS=linux GOARCH=amd64 go build -a -o bin/tuplectl-linux-amd64 .
  GOOS=darwin go get github.com/zalando/go-keyring
  GOOS=darwin GOARCH=amd64 go build -a -o bin/tuplectl-darwin-amd64 .
else
  go get github.com/zalando/go-keyring
  go build -a -o tuplectl .
fi

