# Node Heartbeat Stage With Lease

This Stage configures the node heartbeat with lease.

Some controllers rely on node conditions, so we synchronize the node information periodically, just like Kubelet.

The `node-heartbeat-with-lease` Stage is applied to nodes that have the `Ready` condition set to `True` in their `status.conditions` field.
When applied, this Stage maintains the `status.conditions`, `status.addresses` and `status.daemonEndpoints` fields for the node.
