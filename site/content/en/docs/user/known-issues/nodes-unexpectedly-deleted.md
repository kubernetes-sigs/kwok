---
title: Nodes Unexpectedly Deleted
---

# Nodes Unexpectedly Deleted

{{< hint "info" >}}

This document walks you through the known issues related to nodes being unexpectedly deleted in KWOK.

{{< /hint >}}

## Cloud Provider Integration

When using KWOK in clusters with existing cloud provider integrations (AWS, GCP, Azure), nodes may be unexpectedly deleted due to interactions with cloud provider controllers that manage node lifecycle.

### Root Causes

- Nodes missing cloud provider identifiers (`providerID` field)
- Cloud provider controllers clean up orphaned node
  - During the brief window between node creation and KWOK management initialization

### Recommended Solution

Add a dummy `providerID` to node specifications:
```yaml
spec:
  providerID: "kwok://<node-name>"
```
This format is recognized as valid but won’t match any real cloud provider logic.
