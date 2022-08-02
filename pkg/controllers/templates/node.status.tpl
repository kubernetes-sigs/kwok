{{ with .status }}

addresses:
{{ with .addresses }}
{{ YAML . 1 }}
{{ else }}
- address: {{ NodeIP }}
  type: InternalIP
{{ end }}

allocatable:
{{ with .allocatable }}
{{ YAML . 1 }}
{{ else }}
  cpu: 1k
  memory: 1Ti
  pods: 1M
{{ end }}

capacity:
{{ with .capacity }}
{{ YAML . 1 }}
{{ else }}
  cpu: 1k
  memory: 1Ti
  pods: 1M
{{ end }}

{{ with .nodeInfo }}
nodeInfo:
  architecture: {{ with .architecture }} {{ . }} {{ else }} "amd64" {{ end }}
  bootID: {{ with .bootID }} {{ . }} {{ else }} "" {{ end }}
  containerRuntimeVersion: {{ with .containerRuntimeVersion }} {{ . }} {{ else }} "" {{ end }}
  kernelVersion: {{ with .kernelVersion }} {{ . }} {{ else }} "" {{ end }}
  kubeProxyVersion: {{ with .kubeProxyVersion }} {{ . }} {{ else }} "fake" {{ end }}
  kubeletVersion: {{ with .kubeletVersion }} {{ . }} {{ else }} "fake" {{ end }}
  machineID: {{ with .machineID }} {{ . }} {{ else }} "" {{ end }}
  operatingSystem: {{ with .operatingSystem }} {{ . }} {{ else }} "linux" {{ end }}
  osImage: {{ with .osImage }} {{ . }} {{ else }} "" {{ end }}
  systemUUID: {{ with .osImage }} {{ . }} {{ else }} "" {{ end }}
{{ end }}

phase: Running

{{ end }}
