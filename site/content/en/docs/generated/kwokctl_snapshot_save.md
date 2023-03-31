## kwokctl snapshot save

Save the snapshot of the cluster

```
kwokctl snapshot save [flags]
```

### Options

```
      --filter strings   Filter the resources to save, only support for k8s format (default [configmap,endpoints,namespace,node,persistentvolumeclaim,persistentvolume,pod,secret,serviceaccount,service,daemonset.apps,deployment.apps,replicaset.apps,statefulset.apps,cronjob.batch,job.batch])
      --format string    Format of the snapshot file (etcd, k8s) (default "etcd")
  -h, --help             help for save
      --path string      Path to the snapshot
```

### Options inherited from parent commands

```
  -c, --config stringArray   config path (default [~/.kwok/kwok.yaml])
      --name string          cluster name (default "kwok")
  -v, --v log-level          number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl snapshot](kwokctl_snapshot.md)	 - Snapshot [save, restore] one of cluster

