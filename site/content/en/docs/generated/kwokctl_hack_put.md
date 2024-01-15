## kwokctl hack put

put data in etcd

```
kwokctl hack put [resource] [name] [flags]
```

### Options

```
  -h, --help               help for put
  -n, --namespace string   namespace of resource
  -o, --output string      output format. One of: (key, none). (default "key")
      --path string        path of the file
      --prefix string      prefix of the key (default "/registry")
```

### Options inherited from parent commands

```
  -c, --config strings   config path (default [~/.kwok/kwok.yaml])
      --dry-run          Print the command that would be executed, but do not execute it
      --name string      cluster name (default "kwok")
  -v, --v log-level      number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl hack](kwokctl_hack.md)	 - [experimental] Hack [get, put, delete] resources in etcd without apiserver

