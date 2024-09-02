## kwok

kwok is a tool for simulating the lifecycle of fake nodes, pods, and other Kubernetes API resources.

```
kwok [flags]
```

### Options

```
      --cidr string                        CIDR of the pod ip (default "10.0.0.1/24")
  -c, --config strings                     config path (default [~/.kwok/kwok.yaml])
      --enable-crds strings                List of CRDs to enable
  -h, --help                               help for kwok
      --kubeconfig string                  Path to the kubeconfig file to use (default "~/.kube/config")
      --manage ManagesSelectorSlice        Manages resources
      --master string                      The address of the Kubernetes API server (overrides any value in kubeconfig).
      --node-ip string                     IP of the node
      --node-lease-duration-seconds uint   Duration of node lease seconds
      --node-name string                   Name of the node
      --node-port int                      Port of the node
      --server-address string              Address to expose the server on
      --tls-cert-file string               File containing the default x509 Certificate for HTTPS
      --tls-private-key-file string        File containing the default x509 private key matching --tls-cert-file
  -v, --v log-level                        number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

