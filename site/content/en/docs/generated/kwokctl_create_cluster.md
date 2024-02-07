## kwokctl create cluster

Creates a cluster

```
kwokctl create cluster [flags]
```

### Options

```
      --controller-port uint32                       Port of kwok-controller given to the host
      --dashboard-image string                       Image of dashboard, only for docker/podman/nerdctl/kind/kind-podman runtime
                                                     '${KWOK_DASHBOARD_IMAGE_PREFIX}/dashboard:${KWOK_DASHBOARD_VERSION}'
                                                      (default "docker.io/kubernetesui/dashboard:v2.7.0")
      --dashboard-port uint32                        Port of dashboard given to the host
      --disable-kube-controller-manager              Disable the kube-controller-manager
      --disable-kube-scheduler                       Disable the kube-scheduler
      --disable-qps-limits                           Disable QPS limits for components
      --enable-crds strings                          List of CRDs to enable
      --enable-metrics-server                        Enable the metrics-server
      --etcd-binary string                           Binary of etcd, only for binary runtime (default "https://github.com/etcd-io/etcd/releases/download/v3.5.11/etcd-v3.5.11-linux-amd64.tar.gz#etcd")
      --etcd-image string                            Image of etcd, only for docker/podman/nerdctl runtime
                                                     '${KWOK_KUBE_IMAGE_PREFIX}/etcd:${KWOK_ETCD_VERSION}'
                                                      (default "registry.k8s.io/etcd:3.5.11-0")
      --etcd-port uint32                             Port of etcd given to the host. The behavior is unstable for kind/kind-podman runtime and may be modified in the future
      --etcd-prefix string                           prefix of the key (default "/registry")
      --heartbeat-factor float                       Scale factor for all about heartbeat (default 5)
  -h, --help                                         help for cluster
      --jaeger-binary string                         Binary of Jaeger, only for binary runtime (default "https://github.com/jaegertracing/jaeger/releases/download/v1.53.0/jaeger-1.53.0-linux-amd64.tar.gz#jaeger-all-in-one")
      --jaeger-image string                          Image of Jaeger, only for docker/podman/nerdctl/kind/kind-podman runtime
                                                     '${KWOK_JAEGER_IMAGE_PREFIX}/all-in-one:${KWOK_JAEGER_VERSION}'
                                                      (default "docker.io/jaegertracing/all-in-one:1.53.0")
      --jaeger-port uint32                           Port to expose Jaeger UI
      --kind-binary string                           Binary of kind, only for kind/kind-podman runtime
                                                      (default "https://github.com/kubernetes-sigs/kind/releases/download/v0.19.0/kind-linux-amd64")
      --kind-node-image string                       Image of kind node, only for kind/kind-podman runtime
                                                     '${KWOK_KIND_NODE_IMAGE_PREFIX}/node:${KWOK_KUBE_VERSION}'
                                                      (default "docker.io/kindest/node:v1.29.0")
      --kube-admission                               Enable admission for kube-apiserver, only for non kind/kind-podman runtime (default true)
      --kube-apiserver-binary string                 Binary of kube-apiserver, only for binary runtime
                                                      (default "https://dl.k8s.io/release/v1.29.0/bin/linux/amd64/kube-apiserver")
      --kube-apiserver-cors-allowed-origin strings   List of origins that are allowed to perform CORS requests
      --kube-apiserver-image string                  Image of kube-apiserver, only for docker/podman/nerdctl runtime
                                                     '${KWOK_KUBE_IMAGE_PREFIX}/kube-apiserver:${KWOK_KUBE_VERSION}'
                                                      (default "registry.k8s.io/kube-apiserver:v1.29.0")
      --kube-apiserver-port uint32                   Port of the apiserver (default random)
      --kube-audit-policy string                     Path to the file that defines the audit policy configuration
      --kube-authorization                           Enable authorization for kube-apiserver, only for non kind/kind-podman runtime (default true)
      --kube-controller-manager-binary string        Binary of kube-controller-manager, only for binary runtime
                                                      (default "https://dl.k8s.io/release/v1.29.0/bin/linux/amd64/kube-controller-manager")
      --kube-controller-manager-image string         Image of kube-controller-manager, only for docker/podman/nerdctl runtime
                                                     '${KWOK_KUBE_IMAGE_PREFIX}/kube-controller-manager:${KWOK_KUBE_VERSION}'
                                                      (default "registry.k8s.io/kube-controller-manager:v1.29.0")
      --kube-controller-manager-port uint32          Port of kube-controller-manager given to the host, only for binary and docker/podman/nerdctl runtime
      --kube-feature-gates string                    A set of key=value pairs that describe feature gates for alpha/experimental features of Kubernetes
      --kube-runtime-config string                   A set of key=value pairs that enable or disable built-in APIs
      --kube-scheduler-binary string                 Binary of kube-scheduler, only for binary runtime
                                                      (default "https://dl.k8s.io/release/v1.29.0/bin/linux/amd64/kube-scheduler")
      --kube-scheduler-config string                 Path to a kube-scheduler configuration file
      --kube-scheduler-image string                  Image of kube-scheduler, only for docker/podman/nerdctl runtime
                                                     '${KWOK_KUBE_IMAGE_PREFIX}/kube-scheduler:${KWOK_KUBE_VERSION}'
                                                      (default "registry.k8s.io/kube-scheduler:v1.29.0")
      --kube-scheduler-port uint32                   Port of kube-scheduler given to the host, only for binary and docker/podman/nerdctl runtime
      --kubeconfig string                            The path to the kubeconfig file will be added to the newly created cluster and set to current-context (default "~/.kube/config")
      --kwok-controller-binary string                Binary of kwok-controller, only for binary runtime
                                                      (default "https://github.com/kubernetes-sigs/kwok/releases/download/v0.6.0/kwok-linux-amd64")
      --kwok-controller-image string                 Image of kwok-controller, only for docker/podman/nerdctl/kind/kind-podman runtime
                                                     '${KWOK_IMAGE_PREFIX}/kwok:${KWOK_VERSION}'
                                                      (default "registry.k8s.io/kwok/kwok:v0.6.0")
      --metrics-server-binary string                 Binary of metrics-server, only for binary runtime (default "https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.7.0/metrics-server-linux-amd64")
      --metrics-server-image string                  Image of metrics-server, only for docker/podman/nerdctl/kind/kind-podman runtime
                                                     '${KWOK_METRICS_SERVER_IMAGE_PREFIX}/metrics-server:${KWOK_METRICS_SERVER_VERSION}'
                                                      (default "registry.k8s.io/metrics-server/metrics-server:v0.7.0")
      --node-lease-duration-seconds uint             Duration of node lease in seconds (default 40)
      --prometheus-binary string                     Binary of Prometheus, only for binary runtime (default "https://github.com/prometheus/prometheus/releases/download/v2.49.1/prometheus-2.49.1.linux-amd64.tar.gz#prometheus")
      --prometheus-image string                      Image of Prometheus, only for docker/podman/nerdctl/kind/kind-podman runtime
                                                     '${KWOK_PROMETHEUS_IMAGE_PREFIX}/prometheus:${KWOK_PROMETHEUS_VERSION}'
                                                      (default "docker.io/prom/prometheus:v2.49.1")
      --prometheus-port uint32                       Port to expose Prometheus metrics
      --quiet-pull                                   Pull without printing progress information
      --runtime string                               Runtime of the cluster (binary or docker or kind or kind-podman or nerdctl or podman)
      --secure-port                                  The apiserver port on which to serve HTTPS with authentication and authorization, is not available before Kubernetes 1.13.0 (default true)
      --timeout duration                             Timeout for waiting for the cluster to be created
      --wait duration                                Wait for the cluster to be ready
```

### Options inherited from parent commands

```
  -c, --config strings   config path (default [~/.kwok/kwok.yaml])
      --dry-run          Print the command that would be executed, but do not execute it
      --name string      cluster name (default "kwok")
  -v, --v log-level      number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl create](kwokctl_create.md)	 - Creates one of [cluster]

