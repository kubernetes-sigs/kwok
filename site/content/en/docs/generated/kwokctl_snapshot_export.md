## kwokctl snapshot export

[experimental] Export the snapshots of external clusters

```
kwokctl snapshot export [flags]
```

### Options

```
      --as string           Username to impersonate for the operation. User could be a regular user or a service account in a namespace.
      --as-group strings    Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --filter strings      Filter the resources to export (default [namespace,node,serviceaccount,configmap,secret,limitrange,runtimeclass.node.k8s.io,priorityclass.scheduling.k8s.io,clusterrolebindings.rbac.authorization.k8s.io,clusterroles.rbac.authorization.k8s.io,rolebindings.rbac.authorization.k8s.io,roles.rbac.authorization.k8s.io,daemonset.apps,deployment.apps,replicaset.apps,statefulset.apps,cronjob.batch,job.batch,persistentvolumeclaim,persistentvolume,pod,service,endpoints])
  -h, --help                help for export
      --kubeconfig string   Path to the kubeconfig file to use
      --path string         Path to the snapshot
```

### Options inherited from parent commands

```
  -c, --config stringArray   config path (default [~/.kwok/kwok.yaml])
      --name string          cluster name (default "kwok")
  -v, --v log-level          number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl snapshot](kwokctl_snapshot.md)	 - Snapshot [save, restore, export] one of cluster

