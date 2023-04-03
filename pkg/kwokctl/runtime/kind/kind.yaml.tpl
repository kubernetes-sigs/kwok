kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4

networking:
  apiServerAddress: "0.0.0.0"
{{ if .KubeApiserverPort }}
  apiServerPort: {{ .KubeApiserverPort }}
{{ end }}
nodes:
- role: control-plane

{{ if or .PrometheusPort .KwokControllerPort .EtcdPort }}
  extraPortMappings:
  {{ if .PrometheusPort }}
  - containerPort: 9090
    hostPort: {{ .PrometheusPort }}
    listenAddress: "0.0.0.0"
    protocol: TCP
  {{ end }}
  {{ if .KwokControllerPort }}
  - containerPort: 10247
    hostPort: {{ .KwokControllerPort }}
    listenAddress: "0.0.0.0"
    protocol: TCP
  {{ end }}
  {{ if .EtcdPort }}
  - containerPort: 2379
    hostPort: {{ .EtcdPort }}
    listenAddress: "0.0.0.0"
    protocol: TCP
  {{ end }}
{{ end }}

  kubeadmConfigPatches:

{{ if or .EtcdExtraArgs .EtcdExtraVolumes }}
  - |
    kind: ClusterConfiguration
    etcd:
      local:
      {{ if .EtcdExtraArgs }}
        extraArgs:
        {{ range .EtcdExtraArgs }}
          "{{.Key}}": "{{.Value}}"
        {{ end }}
      {{ end }}

      {{ if .EtcdExtraVolumes }}
        extraVolumes:
        {{ range .EtcdExtraVolumes }}
        - name: {{ .Name }}
          hostPath: /var/components/etcd{{ .MountPath }}
          mountPath: {{ .MountPath }}
          readOnly: {{ .ReadOnly }}
          pathType: {{ .PathType }}
        {{ end }}
      {{ end }}

{{ end }}

{{ if or .ApiserverExtraArgs .ApiserverExtraVolumes }}
  - |
    kind: ClusterConfiguration
    apiServer:
    {{ if .ApiserverExtraArgs }}
      extraArgs:
      {{ range .ApiserverExtraArgs }}
        "{{.Key}}": "{{.Value}}"
      {{ end }}
    {{ end }}

    {{ if .ApiserverExtraVolumes }}
      extraVolumes:
      {{ range .ApiserverExtraVolumes }}
      - name: {{ .Name }}
        hostPath: /var/components/apiserver{{ .MountPath }}
        mountPath: {{ .MountPath }}
        readOnly: {{ .ReadOnly }}
        pathType: {{ .PathType }}
      {{ end }}
    {{ end }}
{{ end }}

{{ if or .ControllerManagerExtraArgs .ControllerManagerExtraVolumes }}
  - |
    kind: ClusterConfiguration
    controllerManager:
    {{ if .ControllerManagerExtraArgs }}
      extraArgs:
      {{ range .ControllerManagerExtraArgs }}
        "{{.Key}}": "{{.Value}}"
      {{ end }}
    {{ end }}

    {{ if .ControllerManagerExtraVolumes }}
      extraVolumes:
      {{ range .ControllerManagerExtraVolumes }}
      - name: {{ .Name }}
        hostPath: /var/components/controller-manager{{ .MountPath }}
        mountPath: {{ .MountPath }}
        readOnly: {{ .ReadOnly }}
        pathType: {{ .PathType }}
      {{ end }}
    {{ end }}
{{ end }}

{{ if or .SchedulerExtraArgs .SchedulerExtraVolumes }}
  - |
    kind: ClusterConfiguration
    scheduler:
    {{ if .SchedulerExtraArgs }}
      extraArgs:
      {{ range .SchedulerExtraArgs }}
        "{{.Key}}": "{{.Value}}"
      {{ end }}
    {{ end }}

    {{ if .SchedulerExtraVolumes }}
      extraVolumes:
      {{ range .SchedulerExtraVolumes }}
      - name: {{ .Name }}
        hostPath: /var/components/scheduler{{ .MountPath }}
        mountPath: {{ .MountPath }}
        readOnly: {{ .ReadOnly }}
        pathType: {{ .PathType }}
      {{ end }}
    {{ end }}
{{ end }}

  # mount the local file on the control plane
  extraMounts:
  - hostPath: {{ .ConfigPath }}
    containerPath: /etc/kwok/kwok.yaml
    readOnly: true

{{ range .EtcdExtraVolumes }}
  - hostPath: {{ .HostPath }}
    containerPath: /var/components/etcd{{ .MountPath }}
    readOnly: {{ .ReadOnly }}
{{ end }}

{{ range .ApiserverExtraVolumes }}
  - hostPath: {{ .HostPath }}
    containerPath: /var/components/apiserver{{ .MountPath }}
    readOnly: {{ .ReadOnly }}
{{ end }}

{{ range .ControllerManagerExtraVolumes }}
  - hostPath: {{ .HostPath }}
    containerPath: /var/components/controller-manager{{ .MountPath }}
    readOnly: {{ .ReadOnly }}
{{ end }}

{{ range .SchedulerExtraVolumes }}
  - hostPath: {{ .HostPath }}
    containerPath: /var/components/scheduler{{ .MountPath }}
    readOnly: {{ .ReadOnly }}
{{ end }}

{{ if .FeatureGates }}
featureGates:
{{ range .FeatureGates }}
  - {{ . }}
{{ end }}
{{ end }}

{{ if .RuntimeConfig }}
runtimeConfig:
{{ range .RuntimeConfig }}
  - {{ . }}
{{ end }}
{{ end }}
