apiVersion: v1
kind: Pod
metadata:
  name: dashboard
  namespace: kube-system
  labels:
    app: dashboard
spec:
  containers:
  - name: dashboard
    image: {{ .DashboardImage }}
    {{ with .ExtraEnvs }}
    env:
    {{ range . }}
    - name: {{ .Name }}
      value: {{ .Value }}
    {{ end }}
    {{ end }}
    args:
    - --kubeconfig=/etc/kubernetes/admin.conf
    - --insecure-bind-address=0.0.0.0
    - --insecure-port=8000
    - --bind-address=127.0.0.1
    - --port=0
    - --enable-insecure-login
    - --enable-skip-login
    - --disable-settings-authorizer
    - --metrics-provider=none
    - --namespace=kube-system
    - --system-banner={{ .Banner }}
    {{ range .ExtraArgs }}
    - --{{ .Key }}={{ .Value }}
    {{ end }}
    volumeMounts:
    - mountPath: /etc/kubernetes/admin.conf
      name: kubeconfig
      readOnly: true
    - mountPath: /etc/kubernetes/pki
      name: k8s-certs
      readOnly: true
    {{ range .ExtraVolumes }}
    - mountPath: {{ .MountPath }}
      name: {{ .Name }}
      readOnly: {{ .ReadOnly }}
    {{ end }}
    securityContext:
      privileged: true
      runAsUser: 0
      runAsGroup: 0
  restartPolicy: Always
  hostNetwork: true
  nodeName: {{ .Name }}-control-plane
  volumes:
  - hostPath:
      path: /etc/kubernetes/admin.conf
      type: FileOrCreate
    name: kubeconfig
  - hostPath:
      path: /etc/kubernetes/pki
      type: DirectoryOrCreate
    name: k8s-certs
  {{ range .ExtraVolumes }}
  - hostPath:
      path: /var/components/controller{{ .MountPath }}
      type: {{ .PathType }}
    name: {{ .Name }}
  {{ end }}
