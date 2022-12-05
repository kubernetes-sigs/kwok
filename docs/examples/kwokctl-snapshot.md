# Kwokctl Save/Restore a Cluster

This document walks you through how to save and restore a cluster with `kwokctl`

## Save cluster data to file

``` bash
kwokctl snapshot save --path snapshot.db
```

## Restore cluster data from file

``` bash
kwokctl snapshot restore --path snapshot.db
```
