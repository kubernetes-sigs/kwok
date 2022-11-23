{{ $scheduleTime := Schedule . }}
{{ $finish := Finish . "last" $scheduleTime }}
{{ if not $finish }}
conditions:
- lastTransitionTime: {{ $scheduleTime }}
  status: "True"
  type: Initialized
- lastTransitionTime: {{ $scheduleTime }}
  status: "True"
  type: Ready
- lastTransitionTime: {{ $scheduleTime }}
  status: "True"
  type: ContainersReady
- lastTransitionTime: {{ $scheduleTime }}
  status: "True"
  type: PodScheduled
{{ range .spec.readinessGates }}
- lastTransitionTime: {{ $scheduleTime }}
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
      startedAt: {{ $scheduleTime }}
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
      finishedAt: {{ $scheduleTime }}
      reason: Completed
      startedAt: {{ $scheduleTime }}
{{ end }}

{{ with .status }}
hostIP: {{ with .hostIP }} {{ . }} {{ else }} {{ NodeIP }} {{ end }}
podIP: {{ with .podIP }} {{ . }} {{ else }} {{ PodIP }} {{ end }}
{{ end }}

phase: Running
startTime: {{ $scheduleTime }}
{{- else}}
{{ $endTime := LastTime . "last" $scheduleTime}}
conditions:
- lastTransitionTime: {{ $endTime }}
  status: "True"
  type: Initialized
  reason: PodCompleted
- lastTransitionTime: {{ $endTime }}
  reason: PodCompleted
  status: "False"
  type: Ready
- lastTransitionTime: {{ $endTime }}
  reason: PodCompleted
  status: "False"
  type: ContainersReady
- lastTransitionTime: {{ $scheduleTime }}
  status: "True"
  type: PodScheduled
{{ range .spec.readinessGates }}
- lastTransitionTime: {{ $endTime }}
  status: "True"
  type: {{ .conditionType }}
{{ end }}

containerStatuses:
{{ range .spec.containers }}
- image: {{ .image }}
  name: {{ .name }}
  ready: false
  restartCount: 0
  started: false
  state:
    terminated:
      exitCode: 0
      finishedAt: {{ $endTime }}
      reason: Completed
      startedAt: {{ $scheduleTime }}
{{ end }}

{{ with .status }}
hostIP: {{ with .hostIP }} {{ . }} {{ else }} {{ NodeIP }} {{ end }}
podIP: {{ with .podIP }} {{ . }} {{ else }} {{ PodIP }} {{ end }}
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
      finishedAt: {{ $scheduleTime }}
      reason: Completed
      startedAt: {{ $scheduleTime }}
{{ end }}

phase: Succeeded
startTime: {{ $scheduleTime }}
{{- end}}