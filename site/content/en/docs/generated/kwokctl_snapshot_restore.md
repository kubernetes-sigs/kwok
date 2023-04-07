## kwokctl snapshot restore

Restore the snapshot of the cluster

```
kwokctl snapshot restore [flags]
```

### Options

```
      --filter strings   Filter the resources to restore, only support for k8s format (default [namespace,node,serviceaccount,configmap,secret,daemonset.apps,deployment.apps,replicaset.apps,statefulset.apps,cronjob.batch,job.batch,persistentvolumeclaim,persistentvolume,pod,service,endpoints])
      --format string    Format of the snapshot file (etcd, k8s) (default "etcd")
  -h, --help             help for restore
      --path string      Path to the snapshot
```

### Options inherited from parent commands

```
  -c, --config stringArray   config path (default [~/.kwok/kwok.yaml])
      --name string          cluster name (default "kwok")
  -v, --v log-level          number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl snapshot](kwokctl_snapshot.md)	 - Snapshot [save, restore, export] one of cluster

