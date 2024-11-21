#!/bin/bash

KWOK_ROOT_DIR=~/kubefuture/kwok

# check if we have login docker
docker login

pushd $KWOK_ROOT_DIR

IMAGE_PREFIX=localhost BUILDER=docker make build build-image
# output of docker images
# localhost/kwok                       v0.6.0-126-gf972843-dirty                        399657af022f   About a minute ago   98.9MB

imageTag=$(git describe --tags --dirty)
echo "imageTag: ${imageTag}"
echo "retag it as docker.io/jokerwyt/kwok:latest"

# retag it as docker.io/jokerwyt/kwok:latest
docker tag localhost/kwok:${imageTag} docker.io/jokerwyt/kwok:latest

# push
docker push docker.io/jokerwyt/kwok:latest

popd