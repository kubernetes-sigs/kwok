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
    - --server-address=0.0.0.0:8080
    - --kubeconfig=/etc/kubernetes/admin.conf
    - --node-ip=$(POD_IP)
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
        port: 8080
        scheme: HTTP
      initialDelaySeconds: 2
      periodSeconds: 10
      timeoutSeconds: 2
    name: kwok-controller
    readinessProbe:
      failureThreshold: 5
      httpGet:
        path: /healthz
        port: 8080
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
