## kwokctl scale

Scale a resource in cluster

```
kwokctl scale [node, pod, ...] [name] [flags]
```

### Options

```
  -h, --help                help for scale
  -n, --namespace string    Namespace of resource to scale
      --param stringArray   Parameter to update
      --replicas int        Number of replicas (default 1)
      --serial-length int   Length of serial number (default 6)
```

### Options inherited from parent commands

```
  -c, --config strings   config path (default [~/.kwok/kwok.yaml])
      --dry-run          Print the command that would be executed, but do not execute it
      --name string      cluster name (default "kwok")
  -v, --v log-level      number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl](kwokctl.md)	 - kwokctl is a tool to streamline the creation and management of clusters, with nodes simulated by kwok

