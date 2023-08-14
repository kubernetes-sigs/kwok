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
    {{ with .ExtraEnvs }}
    volumeMounts:
    {{ range . }}
    - mountPath: {{ .MountPath }}
      name: {{ .Name }}
      readOnly: {{ .ReadOnly }}
    {{ end }}
    {{ end }}
  restartPolicy: Always
  hostNetwork: true
  nodeName: {{ .Name }}-control-plane
  {{ range .ExtraVolumes }}
  volumes:
  {{ range . }}
  - hostPath:
      path: /var/components/controller{{ .MountPath }}
      type: {{ .PathType }}
    name: {{ .Name }}
  {{ end }}
  {{ end }}
