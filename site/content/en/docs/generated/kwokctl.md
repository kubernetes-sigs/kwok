## kwokctl

kwokctl is a tool to streamline the creation and management of clusters, with nodes simulated by kwok

```
kwokctl [command] [flags]
```

### Options

```
  -c, --config stringArray   config path (default [~/.kwok/kwok.yaml])
  -h, --help                 help for kwokctl
      --name string          cluster name (default "kwok")
  -v, --v int                number for the log level verbosity
```

### SEE ALSO

* [kwokctl create](kwokctl_create.md)	 - Creates one of [cluster]
* [kwokctl delete](kwokctl_delete.md)	 - Deletes one of [cluster]
* [kwokctl etcdctl](kwokctl_etcdctl.md)	 - etcdctl in cluster
* [kwokctl get](kwokctl_get.md)	 - Gets one of [artifacts, clusters, kubeconfig]
* [kwokctl kubectl](kwokctl_kubectl.md)	 - kubectl in cluster
* [kwokctl logs](kwokctl_logs.md)	 - Logs one of [audit, etcd, kube-apiserver, kube-controller-manager, kube-scheduler, kwok-controller, prometheus]
* [kwokctl snapshot](kwokctl_snapshot.md)	 - Snapshot [save, restore] one of cluster
* [kwokctl start](kwokctl_start.md)	 - Start one of [cluster]
* [kwokctl stop](kwokctl_stop.md)	 - Stop one of [cluster]

