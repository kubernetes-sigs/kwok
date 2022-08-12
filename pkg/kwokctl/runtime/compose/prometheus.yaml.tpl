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
- job_name: "prometheus"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - localhost:9090
- job_name: "etcd"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - "{{ .ProjectName }}-etcd:2379"
- job_name: "kwok-controller"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - "{{ .ProjectName }}-kwok-controller:8080"

{{ if .SecretPort }}
- job_name: "kube-apiserver"
  scheme: https
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  tls_config:
    cert_file: "{{ .AdminCrtPath }}"
    key_file: "{{ .AdminKeyPath }}"
    insecure_skip_verify: true
  static_configs:
  - targets:
    - "{{ .ProjectName }}-kube-apiserver:6443"
- job_name: "kube-controller-manager"
  scheme: https
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  tls_config:
    cert_file: "{{ .AdminCrtPath }}"
    key_file: "{{ .AdminKeyPath }}"
    insecure_skip_verify: true
  static_configs:
  - targets:
    - "{{ .ProjectName }}-kube-controller-manager:10257"
- job_name: "kube-scheduler"
  scheme: https
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  tls_config:
    cert_file: "{{ .AdminCrtPath }}"
    key_file: "{{ .AdminKeyPath }}"
    insecure_skip_verify: true
  static_configs:
  - targets:
    - "{{ .ProjectName }}-kube-scheduler:10259"
{{ else }}
- job_name: "kube-apiserver"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - "{{ .ProjectName }}-kube-apiserver:8080"
- job_name: "kube-controller-manager"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - "{{ .ProjectName }}-kube-controller-manager:10252"
- job_name: "kube-scheduler"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - "{{ .ProjectName }}-kube-scheduler:10251"
{{ end }}
