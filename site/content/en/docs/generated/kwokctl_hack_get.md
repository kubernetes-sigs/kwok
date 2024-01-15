## kwokctl hack get

get data in etcd

```
kwokctl hack get [resource] [name] [flags]
```

### Options

```
      --chunk-size int     chunk size of the list pager (default 500)
  -h, --help               help for get
  -n, --namespace string   namespace of resource
  -o, --output string      output format. One of: (json, yaml, raw, key). (default "yaml")
      --prefix string      prefix of the key (default "/registry")
  -w, --watch              after listing/getting the requested object, watch for changes
      --watch-only         watch for changes to the requested object(s), without listing/getting first
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

