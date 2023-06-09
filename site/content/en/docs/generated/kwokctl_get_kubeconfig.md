## kwokctl get kubeconfig

Prints cluster kubeconfig

```
kwokctl get kubeconfig [flags]
```

### Options

```
      --group strings              Signing certificate with the specified groups if modified (default [system:masters])
  -h, --help                       help for kubeconfig
      --host string                Override host[:port] for kubeconfig (default "127.0.0.1")
      --insecure-skip-tls-verify   Skip server certificate verification
      --user string                Signing certificate with the specified user if modified (default "kwok-admin")
```

### Options inherited from parent commands

```
  -c, --config stringArray   config path (default [~/.kwok/kwok.yaml])
      --name string          cluster name (default "kwok")
  -v, --v log-level          number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl get](kwokctl_get.md)	 - Gets one of [artifacts, clusters, kubeconfig]

