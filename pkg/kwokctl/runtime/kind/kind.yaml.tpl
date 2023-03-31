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
{{ if or .AuditPolicy .APIServerExtraArgs }}
  - |
    kind: ClusterConfiguration
    apiServer:
      # enable auditing flags on the API server
      extraArgs:
      {{ range .APIServerExtraArgs }}
        "{{.Key}}": "{{.Value}}"
      {{ end }}
      {{ if .AuditPolicy }}
        audit-log-path: /var/log/kubernetes/audit.log
        audit-policy-file: /etc/kubernetes/audit/audit.yaml
      # mount new files / directories on the control plane
      extraVolumes:
      - name: kubernetes
        hostPath: /etc/kubernetes
        mountPath: /etc/kubernetes
        readOnly: true
        pathType: DirectoryOrCreate
      - name: logs
        hostPath: /var/log/kubernetes
        mountPath: /var/log/kubernetes
        readOnly: false
        pathType: DirectoryOrCreate
      {{ end }}
{{ end }}

{{ if .EtcdExtraArgs }}
  - |
    kind: ClusterConfiguration
    etcd:
      local:
        extraArgs:
        {{ range .EtcdExtraArgs }}
          "{{.Key}}": "{{.Value}}"
        {{ end }}
{{ end }}

{{ if .ControllerManagerExtraArgs }}
  - |
    kind: ClusterConfiguration
    controllerManager:
      extraArgs:
      {{ range .ControllerManagerExtraArgs }}
        "{{.Key}}": "{{.Value}}"
      {{ end }}
{{ end }}

{{ if or .SchedulerConfig .SchedulerExtraArgs }}
  - |
    kind: ClusterConfiguration
    scheduler:
      extraArgs:
      {{ range .SchedulerExtraArgs }}
        "{{.Key}}": "{{.Value}}"
      {{ end }}
      {{ if .SchedulerConfig }}
        config: /etc/kubernetes/scheduler/scheduler.yaml
      extraVolumes:
      - name: kubernetes
        hostPath: /etc/kubernetes
        mountPath: /etc/kubernetes
        readOnly: true
        pathType: DirectoryOrCreate
      {{ end }}
{{ end }}

  # mount the local file on the control plane
  extraMounts:
  - hostPath: {{ .ConfigPath }}
    containerPath: /etc/kwok/kwok.yaml
    readOnly: true

{{ if .AuditPolicy }}
  - hostPath: {{ .AuditPolicy }}
    containerPath: /etc/kubernetes/audit/audit.yaml
    readOnly: true
  - hostPath: {{ .AuditLog }}
    containerPath: /var/log/kubernetes/audit.log
    readOnly: false
{{ end }}
{{ if .SchedulerConfig }}
  - hostPath: {{ .SchedulerConfig }}
    containerPath: /etc/kubernetes/scheduler/scheduler.yaml
    readOnly: true
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
