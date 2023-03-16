## kwokctl snapshot restore

Restore the snapshot of the cluster

```
kwokctl snapshot restore [flags]
```

### Options

```
      --filter strings   Filter the resources to restore, only support for k8s format (default [configmap,endpoints,namespace,node,persistentvolumeclaim,persistentvolume,pod,secret,serviceaccount,service,daemonset.apps,deployment.apps,replicaset.apps,statefulset.apps,cronjob.batch,job.batch])
      --format string    Format of the snapshot file (etcd, k8s) (default "etcd")
  -h, --help             help for restore
      --path string      Path to the snapshot
```

### Options inherited from parent commands

```
  -c, --config stringArray   config path (default [~/.kwok/kwok.yaml])
      --name string          cluster name (default "kwok")
  -v, --v int                number for the log level verbosity
```

### SEE ALSO

* [kwokctl snapshot](kwokctl_snapshot.md)	 - Snapshot [save, restore] one of cluster

