## kwokctl snapshot

[experimental] Snapshot [save, restore, record, replay, export] one of cluster

```
kwokctl snapshot [command] [flags]
```

### Options

```
  -h, --help   help for snapshot
```

### Options inherited from parent commands

```
  -c, --config strings   config path (default [~/.kwok/kwok.yaml])
      --dry-run          Print the command that would be executed, but do not execute it
      --name string      cluster name (default "kwok")
  -v, --v log-level      number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl](kwokctl.md)	 - kwokctl is a tool to streamline the creation and management of clusters, with nodes simulated by kwok
* [kwokctl snapshot export](kwokctl_snapshot_export.md)	 - Export the snapshots of external clusters
* [kwokctl snapshot record](kwokctl_snapshot_record.md)	 - Record the recording from the cluster
* [kwokctl snapshot replay](kwokctl_snapshot_replay.md)	 - Replay the recording to the cluster
* [kwokctl snapshot restore](kwokctl_snapshot_restore.md)	 - Restore the snapshot of the cluster
* [kwokctl snapshot save](kwokctl_snapshot_save.md)	 - Save the snapshot of the cluster

