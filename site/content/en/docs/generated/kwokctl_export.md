## kwokctl export

Exports one of [logs]

```
kwokctl export [flags]
```

### Options

```
  -h, --help   help for export
```

### Options inherited from parent commands

```
  -c, --config stringArray   config path (default [~/.kwok/kwok.yaml])
      --dry-run              Print the command that would be executed, but do not execute it
      --name string          cluster name (default "kwok")
  -v, --v log-level          number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl](kwokctl.md)	 - kwokctl is a tool to streamline the creation and management of clusters, with nodes simulated by kwok
* [kwokctl export logs](kwokctl_export_logs.md)	 - Exports logs to a tempdir or [output-dir] if specified

