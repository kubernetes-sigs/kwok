## kwok

kwok is a tool for simulating the lifecycle of fake nodes, pods, and other Kubernetes API resources.

```
kwok [flags]
```

### Options

```
      --cidr string                                    CIDR of the pod ip (default "10.0.0.1/24")
  -c, --config strings                                 config path (default [~/.kwok/kwok.yaml])
      --enable-crds strings                            List of CRDs to enable
  -h, --help                                           help for kwok
      --kubeconfig string                              Path to the kubeconfig file to use (default "~/.kube/config")
      --manage-all-nodes                               All nodes will be watched and managed. It's conflicted with manage-nodes-with-annotation-selector, manage-nodes-with-label-selector and manage-single-node.
      --manage-nodes-with-annotation-selector string   Nodes that match the annotation selector will be watched and managed. It's conflicted with manage-all-nodes and manage-single-node.
      --manage-nodes-with-label-selector string        Nodes that match the label selector will be watched and managed. It's conflicted with manage-all-nodes and manage-single-node.
      --manage-single-node string                      Node that matches the name will be watched and managed. It's conflicted with manage-nodes-with-annotation-selector, manage-nodes-with-label-selector and manage-all-nodes.
      --master string                                  The address of the Kubernetes API server (overrides any value in kubeconfig).
      --node-ip string                                 IP of the node
      --node-lease-duration-seconds uint               Duration of node lease seconds
      --node-name string                               Name of the node
      --node-port int                                  Port of the node
      --server-address string                          Address to expose the server on
      --tls-cert-file string                           File containing the default x509 Certificate for HTTPS
      --tls-private-key-file string                    File containing the default x509 private key matching --tls-cert-file
      --tracing-endpoint string                        Tracing endpoint
      --tracing-sampling-rate-per-million int32        Tracing sampling rate per million
  -v, --v log-level                                    number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

