# Development

This doc walks you through how to run `kwok` and `kwokctl` in the local.

## Building

### Building `kwok` & `kwokctl`

``` bash
make build
```

On `./bin/$(go env GOOS)/$(go env GOARCH)` will contain the freshly build binary upon a successful build.

### Building `kwok` image

On Linux:
``` bash
BINARY=kwok make build
IMAGE_PREFIX=local make image
```

On Darwin:
``` bash
CONTAINER_PLATFORMS=linux/$(go env GOARCH)
BINARY_PLATFORMS=${CONTAINER_PLATFORMS} BINARY=kwok make cross-build
IMAGE_PLATFORMS=${CONTAINER_PLATFORMS} IMAGE_PREFIX=local make cross-image
```

Image `local/kwok:${tag}` will contain in `docker images`

### Start a local cluster with `kwokctl` and `kwok` on Docker

``` bash
KWOK_IMAGE_PREFIX=local ./bin/$(go env GOOS)/$(go env GOARCH)/kwokctl create cluster
```
