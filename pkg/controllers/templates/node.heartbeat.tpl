conditions:
- lastHeartbeatTime: {{ Now }}
  lastTransitionTime: {{ StartTime }}
  message: kubelet is posting ready status
  reason: KubeletReady
  status: "True"
  type: Ready
- lastHeartbeatTime: {{ Now }}
  lastTransitionTime: {{ StartTime }}
  message: kubelet has sufficient disk space available
  reason: KubeletHasSufficientDisk
  status: "False"
  type: OutOfDisk
- lastHeartbeatTime: {{ Now }}
  lastTransitionTime: {{ StartTime }}
  message: kubelet has sufficient memory available
  reason: KubeletHasSufficientMemory
  status: "False"
  type: MemoryPressure
- lastHeartbeatTime: {{ Now }}
  lastTransitionTime: {{ StartTime }}
  message: kubelet has no disk pressure
  reason: KubeletHasNoDiskPressure
  status: "False"
  type: DiskPressure
- lastHeartbeatTime: {{ Now }}
  lastTransitionTime: {{ StartTime }}
  message: RouteController created a route
  reason: RouteCreated
  status: "False"
  type: NetworkUnavailable
