# Stats Summary Metrics (OTel Semantic Conventions)

This Metrics resource provides Prometheus-format metrics using **OpenTelemetry semantic conventions** for Kubernetes resources. The metrics mirror the data typically available from the kubelet's `/stats/summary` endpoint, making it suitable for OpenTelemetry collectors.

The endpoint is available at `/metrics/nodes/{nodeName}/stats/summary`.

## Naming Convention

All metrics follow the [OTel Semantic Conventions for K8s Metrics](https://opentelemetry.io/docs/specs/semconv/system/k8s-metrics/):
- Dot-notation namespaces: `k8s.{resource}.{metric}`
- Labels use `k8s.namespace.name`, `k8s.pod.name`, `k8s.container.name`

## Metrics Provided

### Node-level (`k8s.node.*`)
| Metric | Type | Description |
|--------|------|-------------|
| `k8s.node.cpu.time` | counter | Cumulative CPU time in seconds |
| `k8s.node.cpu.usage` | gauge | Current CPU usage fraction |
| `k8s.node.memory.available` | gauge | Available memory in bytes |
| `k8s.node.memory.usage` | gauge | Memory usage in bytes |
| `k8s.node.memory.rss` | gauge | RSS memory in bytes |
| `k8s.node.memory.working_set` | gauge | Working set memory in bytes |
| `k8s.node.network.io` | counter | Network I/O (with `direction` label) |
| `k8s.node.filesystem.available` | gauge | Available filesystem space |
| `k8s.node.filesystem.capacity` | gauge | Total filesystem capacity |
| `k8s.node.filesystem.usage` | gauge | Filesystem usage |
| `k8s.node.uptime` | gauge | Time since node started |

### Pod-level (`k8s.pod.*`)
| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `k8s.pod.cpu.time` | counter | namespace, pod | Cumulative CPU time |
| `k8s.pod.cpu.usage` | gauge | namespace, pod | Current CPU usage |
| `k8s.pod.memory.available` | gauge | namespace, pod | Available memory |
| `k8s.pod.memory.usage` | gauge | namespace, pod | Memory usage |
| `k8s.pod.memory.rss` | gauge | namespace, pod | RSS memory |
| `k8s.pod.memory.working_set` | gauge | namespace, pod | Working set memory |
| `k8s.pod.memory.major_page_faults` | gauge | namespace, pod | Major page faults ratio |
| `k8s.pod.filesystem.available` | gauge | namespace, pod | Available filesystem bytes |
| `k8s.pod.filesystem.capacity` | gauge | namespace, pod | Filesystem capacity bytes |
| `k8s.pod.filesystem.usage` | gauge | namespace, pod | Filesystem usage bytes |
| `k8s.pod.network.io` | counter | namespace, pod, direction | Network I/O bytes |
| `k8s.pod.network.errors` | counter | namespace, pod, direction | Network errors |
| `k8s.pod.uptime` | gauge | namespace, pod | Time since pod started |
| `k8s.pod.phase` | gauge | namespace, pod, phase | Pod phase (1=Pending, 2=Running, 3=Succeeded, 4=Failed, 5=Unknown) |

### Container-level (`k8s.container.*`)
| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `k8s.container.cpu.time` | counter | namespace, pod, container | Cumulative CPU time |
| `k8s.container.cpu.usage` | gauge | namespace, pod, container | Current CPU usage |
| `k8s.container.cpu.limit` | gauge | namespace, pod, container | CPU limit in cores |
| `k8s.container.cpu.request` | gauge | namespace, pod, container | CPU request in cores |
| `k8s.container.memory.available` | gauge | namespace, pod, container | Available memory |
| `k8s.container.memory.usage` | gauge | namespace, pod, container | Memory usage |
| `k8s.container.memory.rss` | gauge | namespace, pod, container | RSS memory |
| `k8s.container.memory.working_set` | gauge | namespace, pod, container | Working set memory |
| `k8s.container.memory.limit` | gauge | namespace, pod, container | Memory limit in bytes |
| `k8s.container.memory.request` | gauge | namespace, pod, container | Memory request in bytes |
| `k8s.container.filesystem.available` | gauge | namespace, pod, container | Available filesystem |
| `k8s.container.filesystem.capacity` | gauge | namespace, pod, container | Filesystem capacity |
| `k8s.container.filesystem.usage` | gauge | namespace, pod, container | Filesystem usage |
| `k8s.container.uptime` | gauge | namespace, pod, container | Time since container started |
| `k8s.container.ready` | gauge | namespace, pod, container | Container ready status (1/0) |
| `k8s.container.restarts` | counter | namespace, pod, container | Container restart count |

## References

- [OTel Semantic Conventions for K8s Metrics](https://opentelemetry.io/docs/specs/semconv/system/k8s-metrics/)
- [KWOK Metrics Configuration](https://kwok.sigs.k8s.io/docs/user/metrics-configuration)
- [OTel Kubelet Stats Receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/kubeletstatsreceiver)
