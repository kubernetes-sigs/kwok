## kwokctl create cluster

Creates a cluster

### Synopsis

Creates a cluster

```
kwokctl create cluster [flags]
```

### Options

```
      --disable-kube-controller-manager         Disable the kube-controller-manager
      --disable-kube-scheduler                  Disable the kube-scheduler
      --docker-compose-binary string            Binary of Docker-compose, only for docker runtime
                                                 (default "https://github.com/docker/compose/releases/download/v2.13.0/docker-compose-darwin-x86_64")
      --etcd-binary string                      Binary of etcd, only for binary runtime
      --etcd-binary-tar string                  Tar of etcd, if --etcd-binary is set, this is ignored, only for binary runtime
                                                 (default "https://github.com/etcd-io/etcd/releases/download/vunknown/etcd-vunknown-darwin-amd64.zip")
      --etcd-image string                       Image of etcd, only for docker/nerdctl runtime
                                                '${KWOK_KUBE_IMAGE_PREFIX}/etcd:${KWOK_ETCD_VERSION}'
                                                 (default "registry.k8s.io/etcd:unknown")
  -h, --help                                    help for cluster
      --kind-binary string                      Binary of kind, only for kind runtime
                                                 (default "https://github.com/kubernetes-sigs/kind/releases/download/v0.17.0/kind-darwin-amd64")
      --kind-node-image string                  Image of kind node, only for kind runtime
                                                '${KWOK_KIND_NODE_IMAGE_PREFIX}/node:${KWOK_KUBE_VERSION}'
                                                 (default "docker.io/kindest/node:unknown")
      --kube-apiserver-binary string            Binary of kube-apiserver, only for binary runtime
                                                 (default "https://dl.k8s.io/release/unknown/bin/darwin/amd64/kube-apiserver")
      --kube-apiserver-image string             Image of kube-apiserver, only for docker/nerdctl runtime
                                                '${KWOK_KUBE_IMAGE_PREFIX}/kube-apiserver:${KWOK_KUBE_VERSION}'
                                                 (default "registry.k8s.io/kube-apiserver:unknown")
      --kube-apiserver-port uint32              Port of the apiserver (default random)
      --kube-audit-policy string                Path to the file that defines the audit policy configuration
      --kube-authorization                      Enable authorization on secure port
      --kube-controller-manager-binary string   Binary of kube-controller-manager, only for binary runtime
                                                 (default "https://dl.k8s.io/release/unknown/bin/darwin/amd64/kube-controller-manager")
      --kube-controller-manager-image string    Image of kube-controller-manager, only for docker/nerdctl runtime
                                                '${KWOK_KUBE_IMAGE_PREFIX}/kube-controller-manager:${KWOK_KUBE_VERSION}'
                                                 (default "registry.k8s.io/kube-controller-manager:unknown")
      --kube-feature-gates string               A set of key=value pairs that describe feature gates for alpha/experimental features of Kubernetes
      --kube-runtime-config string              A set of key=value pairs that enable or disable built-in APIs
      --kube-scheduler-binary string            Binary of kube-scheduler, only for binary runtime
                                                 (default "https://dl.k8s.io/release/unknown/bin/darwin/amd64/kube-scheduler")
      --kube-scheduler-image string             Image of kube-scheduler, only for docker/nerdctl runtime
                                                '${KWOK_KUBE_IMAGE_PREFIX}/kube-scheduler:${KWOK_KUBE_VERSION}'
                                                 (default "registry.k8s.io/kube-scheduler:unknown")
      --kwok-controller-binary string           Binary of kwok-controller, only for binary runtime
                                                 (default "https://github.com/kubernetes-sigs/kwok/releases/download/unknown/kwok-darwin-amd64")
      --kwok-controller-image string            Image of kwok-controller, only for docker/nerdctl/kind runtime
                                                '${KWOK_IMAGE_PREFIX}/kwok:${KWOK_VERSION}'
                                                 (default "registry.k8s.io/kwok/kwok:unknown")
      --prometheus-binary string                Binary of Prometheus, only for binary runtime
      --prometheus-binary-tar string            Tar of Prometheus, if --prometheus-binary is set, this is ignored, only for binary runtime
                                                 (default "https://github.com/prometheus/prometheus/releases/download/v2.35.0/prometheus-2.35.0.darwin-amd64.tar.gz")
      --prometheus-image string                 Image of Prometheus, only for docker/nerdctl/kind runtime
                                                '${KWOK_PROMETHEUS_IMAGE_PREFIX}/prometheus:${KWOK_PROMETHEUS_VERSION}'
                                                 (default "docker.io/prom/prometheus:v2.35.0")
      --prometheus-port uint32                  Port to expose Prometheus metrics
      --quiet-pull                              Pull without printing progress information
      --runtime string                          Runtime of the cluster () (default "docker")
      --secure-port                             The apiserver port on which to serve HTTPS with authentication and authorization
      --timeout duration                        Timeout for waiting for the cluster to be ready (default 30s)
```

### Options inherited from parent commands

```
      --name string   cluster name (default "kwok")
```

### SEE ALSO

* [kwokctl create](kwokctl_create.md)	 - Creates one of [cluster]

###### Auto generated by spf13/cobra on 14-Dec-2022
