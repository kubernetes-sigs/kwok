# Development

This document provides details on how to build and run `kwok` and `kwokctl` locally.

## Building

### Building `kwok` and `kwokctl`

``` bash
make build
```

On a successful build, the binaries will be located in `./bin/$(go env GOOS)/$(go env GOARCH)`

### Building `kwok` image

```bash
make local-image
```

The image will be tagged as `local/kwok:${tag}` and can be found in `docker images`

### Starting a local cluster with locally built `kwokctl` and `kwok` using Docker

``` bash
./bin/$(go env GOOS)/$(go env GOARCH)/kwokctl create cluster
```
