## kwokctl snapshot replay

Replay the recording to the cluster

```
kwokctl snapshot replay [flags]
```

### Options

```
  -h, --help            help for replay
      --path string     Path to the recording
      --prefix string   prefix of the key (default "/registry")
      --snapshot        Only restore the snapshot
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

