# Development

{{< hint "info" >}}

This document provides details on how to build and run `kwok` and `kwokctl` locally.

{{< /hint >}}

## Directory Structure

- cmd
  - kwok - Main entry point for `kwok`
  - kwokctl - Main entry point for `kwokctl`
- pkg
  - apis - API definitions
    - config
      - v1alpha1 - Configuration API definitions for parsing and converting only
    - internalversion - For all internal use only
    - v1alpha1 - API definitions for parsing and converting only
  - config - Configuration utilities
  - kwok - `kwok` implementation
  - kwokctl - `kwokctl` implementation

## Build and Run

### Start with Containers

Build `kwok` image and `kwokctl` binary.

```bash
IMAGE_PREFIX=localhost BUILDER=docker make build build-image
```

On a successful build, the binaries will be located in `./bin/$(go env GOOS)/$(go env GOARCH)`, the image will be tagged as `localhost/kwok:${tag}` and can be found in `docker images`.

Now, we can create cluster using `kwokctl` with `docker` runtime.

``` bash
./bin/$(go env GOOS)/$(go env GOARCH)/kwokctl create cluster \
  --runtime=docker
```

By the way, you can also use `podman` or `nerdctl` as the builder and the runtime.

### Start with Platform-Specific Binaries

Build `kwok` and `kwokctl` binaries.

``` bash
make build
```

On a successful build, the binaries will be located in `./bin/$(go env GOOS)/$(go env GOARCH)`.

Note that if running in Non-Linux platforms, then you will need to follow build [platform-specific Kubernetes binaries] locally.

Now, we can create cluster using `kwokctl` with `binary` runtime.

``` bash
./bin/$(go env GOOS)/$(go env GOARCH)/kwokctl create cluster \
  --runtime=binary \
  --kwok-controller-binary=./bin/$(go env GOOS)/$(go env GOARCH)/kwok
```

[platform-specific Kubernetes binaries]: {{< relref "/docs/user/kwokctl-platform-specific-binaries" >}}
