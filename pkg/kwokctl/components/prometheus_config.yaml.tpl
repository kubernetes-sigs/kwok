global:
  scrape_interval: 15s
  scrape_timeout: 10s
  evaluation_interval: 15s
alerting:
  alertmanagers:
  - follow_redirects: true
    enable_http2: true
    scheme: http
    timeout: 10s
    api_version: v2
    static_configs:
    - targets: []
scrape_configs:
{{ range .Components }}
{{ $component := . }}
{{ with .MetricsDiscovery }}
- job_name: {{ printf "%s-metrics-discovery" $component.Name | quote }}
  http_sd_configs:
  - url: {{ .Scheme }}://{{ .Host }}{{ .Path }}
{{ end }}
{{ with .Metric }}
- job_name: {{ $component.Name | quote }}
  scheme: {{ .Scheme }}
  honor_timestamps: true
  metrics_path: {{ .Path }}
  follow_redirects: true
  enable_http2: true
{{ if eq .Scheme "https" }}
  tls_config:
    cert_file: {{ .CertPath | quote }}
    key_file: {{ .KeyPath | quote }}
    insecure_skip_verify: {{ .InsecureSkipVerify }}
{{ end }}
  static_configs:
  - targets:
    - {{ .Host }}
{{ end }}
{{ end }}
