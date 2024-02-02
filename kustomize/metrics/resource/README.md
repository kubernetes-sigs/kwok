# Metrics Resource

This Metrics simulates Kubelet's `/metrics/resource` endpoint for ResourceUsage and ClusterResourceUsage only.

In Kubelet, the `/metrics/resource` endpoint is used to expose resource metrics.

In KWOK, all nodes in the simulation exist on the same port.
This requires the use of unique paths for each node on the same port.
Consequently, we use different paths to differentiate the metrics of different nodes.
An example of this would be `/metrics/nodes/{nodeName}/metrics/resource`.

Starting from Metrics-server version 0.7.0,
a new annotation has been added to nodes, namely `metrics.k8s.io/resource-metrics-path`.
The metrics-server uses this annotation to collect the metrics of a given node.

For more information, see [kubernetes-sigs/metrics-server#1253].

[kubernetes-sigs/metrics-server#1253]: https://github.com/kubernetes-sigs/metrics-server/pull/1253
