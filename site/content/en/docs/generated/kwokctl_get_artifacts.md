## kwokctl get artifacts

Lists binaries or images used by cluster

```
kwokctl get artifacts [flags]
```

### Options

```
      --filter string    Filter the list of (binary or image)
  -h, --help             help for artifacts
      --runtime string   Runtime of the cluster (binary or docker or finch or lima or nerdctl or podman)
```

### Options inherited from parent commands

```
  -c, --config strings   config path (default [~/.kwok/kwok.yaml])
      --dry-run          Print the command that would be executed, but do not execute it
      --name string      cluster name (default "kwok")
  -v, --v log-level      number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl get](kwokctl_get.md)	 - Gets one of [artifacts, clusters, components, kubeconfig]

