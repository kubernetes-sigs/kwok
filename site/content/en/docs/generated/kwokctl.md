## kwokctl

kwokctl creates and manages local simulated Kubernetes clusters

```
kwokctl [command] [flags]
```

### Options

```
  -c, --config strings   config path (default [~/.kwok/kwok.yaml])
      --dry-run          print the command that would be executed, but do not execute it
  -h, --help             help for kwokctl
      --name string      cluster name (default "kwok")
  -v, --v log-level      number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl config](kwokctl_config.md)	 - Manages config [convert, reset, tidy, view]
* [kwokctl create](kwokctl_create.md)	 - Creates one of [cluster]
* [kwokctl delete](kwokctl_delete.md)	 - Deletes one of [cluster]
* [kwokctl etcdctl](kwokctl_etcdctl.md)	 - Run etcdctl in cluster
* [kwokctl export](kwokctl_export.md)	 - Exports one of [logs]
* [kwokctl get](kwokctl_get.md)	 - Gets one of [artifacts, clusters, components, kubeconfig]
* [kwokctl kectl](kwokctl_kectl.md)	 - [experimental] Run kubectl-like commands directly against etcd
* [kwokctl kubectl](kwokctl_kubectl.md)	 - Run kubectl in cluster
* [kwokctl logs](kwokctl_logs.md)	 - Logs 'audit' (if enabled) or any component name
* [kwokctl port-forward](kwokctl_port-forward.md)	 - Forward one local ports to a component
* [kwokctl scale](kwokctl_scale.md)	 - Scale a resource in cluster
* [kwokctl snapshot](kwokctl_snapshot.md)	 - [experimental] Snapshot [save, restore, export] one of cluster
* [kwokctl start](kwokctl_start.md)	 - Starts one of [cluster]
* [kwokctl stop](kwokctl_stop.md)	 - Stops one of [cluster]

