kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4

networking:
  apiServerAddress: "0.0.0.0"
{{ if .ApiserverPort }}
  apiServerPort: {{ .ApiserverPort }}
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

{{ if .FeatureGates }}
featureGates:
{{ range .FeatureGates }}
  {{ . }}
{{ end }}
{{ end }}

{{ if .RuntimeConfig }}
runtimeConfig:
{{ range .RuntimeConfig }}
  {{ . }}
{{ end }}
{{ end }}