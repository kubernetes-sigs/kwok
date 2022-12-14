#!/bin/bash

# Get the name of the operating system using the uname command
os_name=$(uname)
set -e

function main() {
  # Check if the operating system is Linux or Darwin
  if [ "$os_name" == "Linux" ]; then
    echo "The operating system is Linux"
    # Build binary and image
    BINARY=kwok make build
    IMAGE_PREFIX=local make image 
  elif [ "$os_name" == "Darwin" ]; then
    echo "The operating system is Darwin (macOS)"
    export CONTAINER_PLATFORMS=linux/$(go env GOARCH) 
    export BINARY_PLATFORMS=${CONTAINER_PLATFORMS} 
    export IMAGE_PLATFORMS=${CONTAINER_PLATFORMS}
    # Cross-build binary and image
    BINARY=kwok make cross-build
    IMAGE_PREFIX=local make cross-image
  else
    echo "The operating system is not Linux or Darwin"
  fi
  # For local use `kwokctl` and `kwok`
  export KWOK_IMAGE_PREFIX=local
  echo "image build complete"
}

main
