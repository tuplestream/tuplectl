#!/usr/bin/env bash

set -euf -o pipefail

BUILDNAME="tuplectl"
IMAGENAME="tuplestream/$BUILDNAME:$(cat VERSION)"
IMAGE_LATEST="tuplestream/$BUILDNAME:latest"

docker build . -t $IMAGENAME --build-arg AUTH0_CLIENT_ID=$AUTH0_CLIENT_ID --build-arg AUTH0_TENANT_URL=$AUTH0_TENANT_URL

if [[ " $@ " =~ " -release" ]]; then
    echo $DKPW | docker login --username $DOCKER_USER --password-stdin
    docker push $IMAGENAME
    docker tag $IMAGENAME $IMAGE_LATEST
    docker push $IMAGE_LATEST
fi
