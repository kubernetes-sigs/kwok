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
  -c, --config strings   config path (default [~/.kwok/kwok.yaml])
      --dry-run          Print the command that would be executed, but do not execute it
      --name string      cluster name (default "kwok")
  -v, --v log-level      number for the log level verbosity (DEBUG, INFO, WARN, ERROR) or (-4, 0, 4, 8) (default INFO)
```

### SEE ALSO

* [kwokctl](kwokctl.md)	 - kwokctl is a tool to streamline the creation and management of clusters, with nodes simulated by kwok
* [kwokctl config convert](kwokctl_config_convert.md)	 - Convert the specified config files to the latest version.
* [kwokctl config reset](kwokctl_config_reset.md)	 - Remove the default config file
* [kwokctl config tidy](kwokctl_config_tidy.md)	 - Tidy the default config file. When combined with --config, it merges the specified configuration files into the default one.
* [kwokctl config view](kwokctl_config_view.md)	 - Display the default config file. When combined with --config, it displays the default config file with the specified ones merged.

