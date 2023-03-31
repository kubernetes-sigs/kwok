kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4

networking:
  apiServerAddress: "0.0.0.0"
{{ if .KubeApiserverPort }}
  apiServerPort: {{ .KubeApiserverPort }}
{{ end }}
nodes:
- role: control-plane

{{ if .PrometheusPort }}
  extraPortMappings:
  - containerPort: 9090
    hostPort: {{ .PrometheusPort }}
    listenAddress: "0.0.0.0"
    protocol: TCP
{{ end }}

  kubeadmConfigPatches:
{{ if .AuditPolicy }}
  - |
    kind: ClusterConfiguration
    apiServer:
      # enable auditing flags on the API server
      extraArgs:
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

{{ if .SchedulerConfig }}
  - |
    kind: ClusterConfiguration
    scheduler:
      extraArgs:
        config: /etc/kubernetes/scheduler/scheduler.yaml
      extraVolumes:
      - name: kubernetes
        hostPath: /etc/kubernetes
        mountPath: /etc/kubernetes
        readOnly: true
        pathType: DirectoryOrCreate
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
