## kwokctl get

Gets one of [artifacts, clusters, components, kubeconfig]

```
kwokctl get [command] [flags]
```

### Options

```
  -h, --help   help for get
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
* [kwokctl get artifacts](kwokctl_get_artifacts.md)	 - Lists binaries or images used by cluster
* [kwokctl get clusters](kwokctl_get_clusters.md)	 - Lists existing clusters by their name
* [kwokctl get components](kwokctl_get_components.md)	 - List components
* [kwokctl get kubeconfig](kwokctl_get_kubeconfig.md)	 - Prints cluster kubeconfig

