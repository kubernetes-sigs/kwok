## kwokctl config tidy

Tidy the default config file. When combined with --config, it merges the specified configuration files into the default one.

```
kwokctl config tidy [flags]
```

### Options

```
  -h, --help   help for tidy
```

### Options inherited from parent commands

```
  -c, --config strings   config path (default [~/.kwok/kwok.yaml])
      --dry-run          Print the command that would be executed, but do not execute it
      --name string      cluster name (default "kwok")
  -v, --v log-level      number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl config](kwokctl_config.md)	 - Manage [reset, tidy, view] default config

