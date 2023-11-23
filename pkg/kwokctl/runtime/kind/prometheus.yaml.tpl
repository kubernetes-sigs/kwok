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
    - targets: [ ]
scrape_configs:
- job_name: "kwok-service-discovery"
  http_sd_configs:
  - url: http://localhost:10247/discovery/prometheus
- job_name: "prometheus"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - "localhost:9090"
- job_name: "etcd"
  scheme: https
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  tls_config:
    cert_file: /etc/kubernetes/pki/apiserver-etcd-client.crt
    key_file: /etc/kubernetes/pki/apiserver-etcd-client.key
    insecure_skip_verify: true
  static_configs:
  - targets:
    - "localhost:2379"
- job_name: "kwok-controller"
  scheme: http
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - "localhost:10247"
- job_name: "kube-apiserver"
  scheme: https
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  tls_config:
    cert_file: /etc/kubernetes/pki/admin.crt
    key_file: /etc/kubernetes/pki/admin.key
    insecure_skip_verify: true
  static_configs:
  - targets:
    - "localhost:6443"
- job_name: "kube-controller-manager"
  scheme: https
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  tls_config:
    cert_file: /etc/kubernetes/pki/admin.crt
    key_file: /etc/kubernetes/pki/admin.key
    insecure_skip_verify: true
  static_configs:
  - targets:
    - "localhost:10257"
- job_name: "kube-scheduler"
  scheme: https
  honor_timestamps: true
  metrics_path: /metrics
  follow_redirects: true
  enable_http2: true
  tls_config:
    cert_file: /etc/kubernetes/pki/admin.crt
    key_file: /etc/kubernetes/pki/admin.key
    insecure_skip_verify: true
  static_configs:
  - targets:
    - "localhost:10259"
