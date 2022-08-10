apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app: kwok-controller
  name: kwok-controller
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    app: kwok-controller
  name: kwok-controller
rules:
- apiGroups:
  - ""
  resources:
  - nodes
  verbs:
  - watch
  - list
  - get
- apiGroups:
  - ""
  resources:
  - nodes/status
  verbs:
  - update
  - patch
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - watch
  - list
  - delete
  - update
  - patch
- apiGroups:
  - ""
  resources:
  - pods/status
  verbs:
  - update
  - patch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    app: kwok-controller
  name: kwok-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kwok-controller
subjects:
- kind: ServiceAccount
  name: kwok-controller
  namespace: kube-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: kwok-controller
  name: kwok-controller
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kwok-controller
  template:
    metadata:
      labels:
        app: kwok-controller
      name: kwok-controller
      namespace: kube-system
    spec:
      containers:
      - args:
        - --manage-all-nodes=false
        - --manage-nodes-with-annotation-selector=kwok.x-k8s.io/node=fake
        - --manage-nodes-with-label-selector=
        - --disregard-status-with-annotation-selector=kwok.x-k8s.io/status=custom
        - --disregard-status-with-label-selector=
        - --server-address=0.0.0.0:8080
        - --cidr=10.0.0.1/24
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
      nodeName: '{{.Name}}-control-plane'
      restartPolicy: Always
      serviceAccount: kwok-controller
      serviceAccountName: kwok-controller
