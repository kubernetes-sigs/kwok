## kwokctl logs

Logs one of [audit, etcd, kube-apiserver, kube-controller-manager, kube-scheduler, kwok-controller, prometheus]

```
kwokctl logs [command] [flags]
```

### Options

```
  -f, --follow   Specify if the logs should be streamed
  -h, --help     help for logs
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

