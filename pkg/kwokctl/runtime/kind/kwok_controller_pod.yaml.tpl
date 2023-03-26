apiVersion: v1
kind: Pod
metadata:
  labels:
    app: kwok-controller
  name: kwok-controller
  namespace: kube-system
spec:
  containers:
  - args:
    - --config=/etc/kwok/kwok.yaml
    - --manage-all-nodes=false
    - --manage-nodes-with-annotation-selector=kwok.x-k8s.io/node=fake
    - --manage-nodes-with-label-selector=
    - --disregard-status-with-annotation-selector=kwok.x-k8s.io/status=custom
    - --disregard-status-with-label-selector=
    - --kubeconfig=/etc/kubernetes/admin.conf
    - --tls-cert-file=/etc/kubernetes/pki/apiserver.crt
    - --tls-private-key-file=/etc/kubernetes/pki/apiserver.key
    - --node-ip=$(POD_IP)
    - --node-name=kwok-controller.kube-system.svc
    - --node-port=10247
    {{ range .ExtraArgs }}
    - {{ . }}
    {{ end }}
    env:
    - name: POD_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    image: '{{.KwokControllerImageName}}:{{.KwokControllerImageTag}}'
    imagePullPolicy: IfNotPresent
    livenessProbe:
      failureThreshold: 3
      httpGet:
        path: /healthz
        port: 10247
        scheme: HTTP
      initialDelaySeconds: 2
      periodSeconds: 10
      timeoutSeconds: 2
    name: kwok-controller
    readinessProbe:
      failureThreshold: 5
      httpGet:
        path: /healthz
        port: 10247
        scheme: HTTP
      initialDelaySeconds: 2
      periodSeconds: 20
      timeoutSeconds: 2
    volumeMounts:
    - mountPath: /etc/kubernetes/admin.conf
      name: kubeconfig
      readOnly: true
    - mountPath: /etc/kwok/kwok.yaml
      name: config
      readOnly: true
    - mountPath: /etc/kubernetes/pki
      name: k8s-certs
      readOnly: true
  hostNetwork: true
  restartPolicy: Always
  volumes:
  - hostPath:
      path: /etc/kubernetes/admin.conf
      type: FileOrCreate
    name: kubeconfig
  - hostPath:
      path: /etc/kwok/kwok.yaml
      type: FileOrCreate
    name: config
  - hostPath:
      path: /etc/kubernetes/pki
      type: DirectoryOrCreate
    name: k8s-certs
