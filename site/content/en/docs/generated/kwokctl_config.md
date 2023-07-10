## kwokctl config

Manage [reset, tidy, view] default config

```
kwokctl config [command] [flags]
```

### Options

```
  -h, --help   help for config
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
* [kwokctl config reset](kwokctl_config_reset.md)	 - Remove the default config file
* [kwokctl config tidy](kwokctl_config_tidy.md)	 - Tidy the default config file with --config
* [kwokctl config view](kwokctl_config_view.md)	 - Display the default config file with --config

