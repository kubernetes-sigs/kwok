{{ $startTime := .metadata.creationTimestamp }}

conditions:
- lastTransitionTime: {{ $startTime }}
  status: "True"
  type: Initialized
- lastTransitionTime: {{ $startTime }}
  status: "True"
  type: Ready
- lastTransitionTime: {{ $startTime }}
  status: "True"
  type: ContainersReady
- lastTransitionTime: {{ $startTime }}
  status: "True"
  type: PodScheduled
{{ range .spec.readinessGates }}
- lastTransitionTime: {{ $startTime }}
  status: "True"
  type: {{ .conditionType }}
{{ end }}

containerStatuses:
{{ range .spec.containers }}
- image: {{ .image }}
  name: {{ .name }}
  ready: true
  restartCount: 0
  state:
    running:
      startedAt: {{ $startTime }}
{{ end }}

initContainerStatuses:
{{ range .spec.initContainers }}
- image: {{ .image }}
  name: {{ .name }}
  ready: true
  restartCount: 0
  state:
    terminated:
      exitCode: 0
      finishedAt: {{ $startTime }}
      reason: Completed
      startedAt: {{ $startTime }}
{{ end }}

{{ with .status }}
hostIP: {{ with .hostIP }} {{ . }} {{ else }} {{ NodeIP }} {{ end }}
podIP: {{ with .podIP }} {{ . }} {{ else }} {{ PodIP }} {{ end }}
{{ end }}

phase: Running
startTime: {{ $startTime }}
