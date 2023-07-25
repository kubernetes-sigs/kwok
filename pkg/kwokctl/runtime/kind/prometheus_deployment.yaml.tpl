apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: prometheus
rules:
  - nonResourceURLs: ["/metrics"]
    verbs: ["get"]
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: prometheus
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prometheus
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prometheus
subjects:
  - kind: ServiceAccount
    name: prometheus
    namespace: kube-system
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-configmap
  namespace: kube-system
data:
  prometheus.yaml: |
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
          insecure_skip_verify: true
        bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
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
          insecure_skip_verify: true
        bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
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
          insecure_skip_verify: true
        bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
        static_configs:
          - targets:
              - "localhost:10259"
---
apiVersion: v1
kind: Pod
metadata:
  name: prometheus
  namespace: kube-system
spec:
  containers:
    - name: prometheus
      image: {{ .PrometheusImage }}
      {{ with .ExtraEnvs }}
      env:
      {{ range . }}
      - name: {{ .Name }}
        value: {{ .Value }}
      {{ end }}
      {{ end }}
      args:
        - --config.file=/etc/prometheus/prometheus.yaml
        {{ if .LogLevel }}
        - --log.level={{ .LogLevel }}
        {{ end }}
        {{ range .ExtraArgs }}
        - --{{ .Key }}={{ .Value }}
        {{ end }}
      ports:
        - name: web
          containerPort: 9090
      securityContext:
        runAsUser: 0
      volumeMounts:
        - name: config-volume
          mountPath: /etc/prometheus/
          readOnly: true
        - mountPath: /etc/kubernetes/pki
          name: k8s-certs
          readOnly: true
        {{ range .ExtraVolumes }}
        - mountPath: {{ .MountPath }}
          name: {{ .Name }}
          readOnly: {{ .ReadOnly }}
        {{ end }}
  volumes:
    - name: config-volume
      configMap:
        name: prometheus-configmap
    - hostPath:
        path: /etc/kubernetes/pki
        type: DirectoryOrCreate
      name: k8s-certs
    {{ range .ExtraVolumes }}
    - hostPath:
        path: {{ .HostPath }}
        type: {{ .PathType }}
      name: {{ .Name }}
    {{ end }}
  serviceAccount: prometheus
  serviceAccountName: prometheus
  restartPolicy: Always
  hostNetwork: true
  nodeName: {{ .Name }}-control-plane
