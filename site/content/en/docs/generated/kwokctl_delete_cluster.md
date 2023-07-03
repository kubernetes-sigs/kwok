## kwokctl delete cluster

Deletes a cluster

```
kwokctl delete cluster [flags]
```

### Options

```
  -h, --help                help for cluster
      --kubeconfig string   The path to the kubeconfig file that will remove the deleted cluster (default "~/.kube/config")
```

### Options inherited from parent commands

```
  -c, --config stringArray   config path (default [~/.kwok/kwok.yaml])
      --dry-run              Print the command that would be executed, but do not execute it
      --name string          cluster name (default "kwok")
  -v, --v log-level          number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl delete](kwokctl_delete.md)	 - Deletes one of [cluster]

