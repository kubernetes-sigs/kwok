## kwokctl

kwokctl is a tool to streamline the creation and management of clusters, with nodes simulated by kwok

```
kwokctl [command] [flags]
```

### Options

```
  -c, --config strings   config path (default [~/.kwok/kwok.yaml])
      --dry-run          Print the command that would be executed, but do not execute it
  -h, --help             help for kwokctl
      --name string      cluster name (default "kwok")
  -v, --v log-level      number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl config](kwokctl_config.md)	 - Manage [reset, tidy, view] default config
* [kwokctl create](kwokctl_create.md)	 - Creates one of [cluster]
* [kwokctl delete](kwokctl_delete.md)	 - Deletes one of [cluster]
* [kwokctl etcdctl](kwokctl_etcdctl.md)	 - etcdctl in cluster
* [kwokctl export](kwokctl_export.md)	 - Exports one of [logs]
* [kwokctl get](kwokctl_get.md)	 - Gets one of [artifacts, clusters, components, kubeconfig]
* [kwokctl hack](kwokctl_hack.md)	 - [experimental] Hack [get, put, delete] resources in etcd without apiserver
* [kwokctl kubectl](kwokctl_kubectl.md)	 - kubectl in cluster
* [kwokctl logs](kwokctl_logs.md)	 - Logs one of [audit, etcd, kube-apiserver, kube-controller-manager, kube-scheduler, kwok-controller, dashboard, metrics-server, prometheus, jaeger]
* [kwokctl port-forward](kwokctl_port-forward.md)	 - Forward one local ports to a component
* [kwokctl scale](kwokctl_scale.md)	 - Scale a resource in cluster
* [kwokctl snapshot](kwokctl_snapshot.md)	 - [experimental] Snapshot [save, restore, record, replay, export] one of cluster
* [kwokctl start](kwokctl_start.md)	 - Start one of [cluster]
* [kwokctl stop](kwokctl_stop.md)	 - Stop one of [cluster]

