# Node Heartbeat Stage (DEPRECATED)

This Stage configures the node heartbeat, only for prior to v0.3.

The `node-heartbeat` Stage is applied to nodes that have the `Ready` condition set to `True` in their `status.conditions` field.
When applied, this Stage maintains the `status.conditions` field for the node.
