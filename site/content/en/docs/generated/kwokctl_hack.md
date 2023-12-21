## kwokctl hack

[experimental] Hack [get, put, delete] resources in etcd without apiserver

```
kwokctl hack [command] [flags]
```

### Options

```
  -h, --help   help for hack
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
* [kwokctl hack delete](kwokctl_hack_delete.md)	 - delete data in etcd
* [kwokctl hack get](kwokctl_hack_get.md)	 - get data in etcd
* [kwokctl hack put](kwokctl_hack_put.md)	 - put data in etcd

