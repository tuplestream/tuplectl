#!/usr/bin/env bash

export GOARCH=amd64
export CGO_ENABLED=0

go build -a -o tuplectl .
