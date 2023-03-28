## kwokctl snapshot restore

Restore the snapshot of the cluster

```
kwokctl snapshot restore [flags]
```

### Options

```
      --format string   Format of the snapshot file (etcd) (default "etcd")
  -h, --help            help for restore
      --path string     Path to the snapshot
```

### Options inherited from parent commands

```
  -c, --config stringArray   config path (default [~/.kwok/kwok.yaml])
      --name string          cluster name (default "kwok")
  -v, --v log-level          number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl snapshot](kwokctl_snapshot.md)	 - Snapshot [save, restore] one of cluster

