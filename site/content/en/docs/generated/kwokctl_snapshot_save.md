## kwokctl snapshot save

Save the snapshot of the cluster

```
kwokctl snapshot save [flags]
```

### Options

```
      --filter strings   Filter the resources to save, only support for k8s format (default [namespace,node,serviceaccount,configmap,secret,limitrange,runtimeclass.node.k8s.io,priorityclass.scheduling.k8s.io,clusterrolebindings.rbac.authorization.k8s.io,clusterroles.rbac.authorization.k8s.io,rolebindings.rbac.authorization.k8s.io,roles.rbac.authorization.k8s.io,daemonset.apps,deployment.apps,replicaset.apps,statefulset.apps,cronjob.batch,job.batch,persistentvolumeclaim,persistentvolume,pod,service,endpoints])
      --format string    Format of the snapshot file (etcd, k8s) (default "etcd")
  -h, --help             help for save
      --path string      Path to the snapshot
```

### Options inherited from parent commands

```
  -c, --config strings   config path (default [~/.kwok/kwok.yaml])
      --dry-run          Print the command that would be executed, but do not execute it
      --name string      cluster name (default "kwok")
  -v, --v log-level      number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl snapshot](kwokctl_snapshot.md)	 - Snapshot [save, restore, record, replay, export] one of cluster

