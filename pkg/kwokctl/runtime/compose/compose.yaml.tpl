version: "3.1"
services:

  # Etcd
  etcd:
    container_name: "{{ .ProjectName }}-etcd"
    image: {{ .EtcdImage }}
    restart: always
    environment:
      - ETCDCTL_API=3
    command:
      - etcd
      - --data-dir
      - {{ .InClusterEtcdDataPath }}
      - --name
      - node0
      - --initial-advertise-peer-urls
      - http://0.0.0.0:2380
      - --listen-peer-urls
      - http://0.0.0.0:2380
      - --advertise-client-urls
      - http://0.0.0.0:2379
      - --listen-client-urls
      - http://0.0.0.0:2379
      - --initial-cluster
      - node0=http://0.0.0.0:2380
      - --auto-compaction-retention
      - "1"
      - --quota-backend-bytes
      - "8589934592"

  # Kube-apiserver
  kube_apiserver:
    container_name: "{{ .ProjectName }}-kube-apiserver"
    image: {{ .KubeApiserverImage }}
    restart: always
    links:
      - etcd
    ports:
{{ if .SecretPort }}
      - {{ .KubeApiserverPort }}:6443
{{ else }}
      - {{ .KubeApiserverPort }}:8080
{{ end }}
    command:
      - kube-apiserver
      - --admission-control
      - ""
      - --etcd-servers
      - http://{{ .ProjectName }}-etcd:2379
      - --etcd-prefix
      - /prefix/registry
      - --allow-privileged
{{ if .RuntimeConfig }}
      - --runtime-config
      - {{ .RuntimeConfig }}
{{ end }}
{{ if .FeatureGates }}
      - --feature-gates
      - {{ .FeatureGates }}
{{ end }}
{{ if .SecretPort }}
      - --authorization-mode
      - Node,RBAC
      - --bind-address
      - 0.0.0.0
      - --secure-port
      - "6443"
      - --tls-cert-file
      - {{ .InClusterAdminCertPath }}
      - --tls-private-key-file
      - {{ .InClusterAdminKeyPath }}
      - --client-ca-file
      - {{ .InClusterCACertPath }}
      - --service-account-key-file
      - {{ .InClusterAdminKeyPath }}
      - --service-account-signing-key-file
      - {{ .InClusterAdminKeyPath }}
      - --service-account-issuer
      - https://kubernetes.default.svc.cluster.local
{{ else }}
      - --insecure-bind-address
      - 0.0.0.0
      - --insecure-port
      - "8080"
{{ end }}

{{ if .AuditPolicy }}
      - --audit-policy-file
      - /etc/kubernetes/audit-policy.yaml
      - --audit-log-path
      - /var/log/kubernetes/audit/audit.log
{{ end }}

{{ if or .SecretPort .AuditPolicy }}
    volumes:
{{ end }}

{{ if .SecretPort }}
      - {{ .AdminKeyPath }}:{{ .InClusterAdminKeyPath }}:ro
      - {{ .AdminCertPath }}:{{ .InClusterAdminCertPath }}:ro
      - {{ .CACertPath }}:{{ .InClusterCACertPath }}:ro
{{ end }}
{{ if .AuditPolicy }}
      - {{ .AuditPolicy }}:/etc/kubernetes/audit-policy.yaml:ro
      - {{ .AuditLog }}:/var/log/kubernetes/audit/audit.log:rw
{{ end }}

  # Kube-controller-manager
  kube_controller_manager:
    container_name: "{{ .ProjectName }}-kube-controller-manager"
    image: {{ .KubeControllerManagerImage }}
    restart: always
    links:
      - kube_apiserver
    command:
      - kube-controller-manager
      - --kubeconfig
      - {{ .InClusterKubeconfigPath }}
{{ if .FeatureGates }}
      - --feature-gates
      - {{ .FeatureGates }}
{{ end }}
{{ if .SecretPort }}
{{ if .PrometheusPath }}
      - --bind-address
      - 0.0.0.0
      - --secure-port
      - "10257"
      - --authorization-always-allow-paths
      - /healthz,/metrics
{{ end }}
      - --root-ca-file
      - {{ .InClusterCACertPath }}
      - --service-account-private-key-file
      - {{ .InClusterAdminKeyPath }}
{{ else }}
{{ if .PrometheusPath }}
      - --address
      - 0.0.0.0
      - --port
      - "10252"
{{ end }}
{{ end }}
    volumes:
      - {{ .KubeconfigPath }}:{{ .InClusterKubeconfigPath }}:ro
{{ if .SecretPort }}
      - {{ .AdminKeyPath }}:{{ .InClusterAdminKeyPath }}:ro
      - {{ .AdminCertPath }}:{{ .InClusterAdminCertPath }}:ro
      - {{ .CACertPath }}:{{ .InClusterCACertPath }}:ro
{{ end }}

  # Kube-scheduler
  kube_scheduler:
    container_name: "{{ .ProjectName }}-kube-scheduler"
    image: {{ .KubeSchedulerImage }}
    restart: always
    links:
      - kube_apiserver
    command:
      - kube-scheduler
      - --kubeconfig
      - {{ .InClusterKubeconfigPath }}
{{ if .FeatureGates }}
      - --feature-gates
      - {{ .FeatureGates }}
{{ end }}
{{ if .PrometheusPath }}
{{ if .SecretPort }}
      - --bind-address
      - 0.0.0.0
      - --secure-port
      - "10259"
      - --authorization-always-allow-paths
      - /healthz,/metrics
{{ else }}
      - --address
      - 0.0.0.0
      - --port
      - "10251"
{{ end }}
{{ end }}
    volumes:
      - {{ .KubeconfigPath }}:{{ .InClusterKubeconfigPath }}:ro
{{ if .SecretPort }}
      - {{ .AdminKeyPath }}:{{ .InClusterAdminKeyPath }}:ro
      - {{ .AdminCertPath }}:{{ .InClusterAdminCertPath }}:ro
      - {{ .CACertPath }}:{{ .InClusterCACertPath }}:ro
{{ end }}

  # Kwok-controller
  kwok_controller:
    container_name: "{{ .ProjectName }}-kwok-controller"
    image: {{ .KwokControllerImage }}
    restart: always
    command:
      - --kubeconfig
      - {{ .InClusterKubeconfigPath }}
      - --cidr
      - 10.0.0.1/24
      - --manage-all-nodes
{{ if .PrometheusPath }}
      - --server-address
      - 0.0.0.0:8080
{{ end }}
    links:
      - kube_apiserver

    volumes:
      - {{ .KubeconfigPath }}:{{ .InClusterKubeconfigPath }}:ro
{{ if .SecretPort }}
      - {{ .AdminKeyPath }}:{{ .InClusterAdminKeyPath }}:ro
      - {{ .AdminCertPath }}:{{ .InClusterAdminCertPath }}:ro
      - {{ .CACertPath }}:{{ .InClusterCACertPath }}:ro
{{ end }}

{{ if .PrometheusPath }}
  # Prometheus
  prometheus:
    container_name: "{{ .ProjectName }}-prometheus"
    image: {{ .PrometheusImage }}
    restart: always
    links:
      - kube_controller_manager
      - kube_scheduler
      - kube_apiserver
      - etcd
      - kwok_controller
    command:
      - --config.file
      - {{ .InClusterPrometheusPath }}
    ports:
      - {{ .PrometheusPort }}:9090
    volumes:
      - {{ .PrometheusPath }}:{{ .InClusterPrometheusPath }}:ro
{{ if .SecretPort }}
      - {{ .AdminKeyPath }}:{{ .InClusterAdminKeyPath }}:ro
      - {{ .AdminCertPath }}:{{ .InClusterAdminCertPath }}:ro
      - {{ .CACertPath }}:{{ .InClusterCACertPath }}:ro
{{ end }}
{{ end }}

# Network
networks:
  default:
    name: {{ .ProjectName }}
