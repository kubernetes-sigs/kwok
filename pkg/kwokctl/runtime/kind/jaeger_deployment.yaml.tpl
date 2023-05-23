apiVersion: v1
kind: Pod
metadata:
  name: jaeger
  namespace: kube-system
  labels:
    app: jaeger
spec:
  containers:
    - name: jaeger
      image: {{ .JaegerImage }}
      {{ with .ExtraEnvs }}
      env:
      {{ range . }}
      - name: {{ .Name }}
        value: {{ .Value }}
      {{ end }}
      {{ end }}
      args:
        - --collector.otlp.enabled=true
        {{ if .LogLevel }}
        - --log-level={{ .LogLevel }}
        {{ end }}
        {{ range .ExtraArgs }}
        - --{{ .Key }}={{ .Value }}
        {{ end }}
  restartPolicy: Always
  hostNetwork: true
  nodeName: {{ .Name }}-control-plane
---
apiVersion: v1
kind: Service
metadata:
  name: jaeger-query
spec:
  selector:
    app: jaeger
  ports:
  - name: query
    port: 16686
    targetPort: 16686
  - name: otlp-grpc
    port: 4317
    targetPort: 4317
