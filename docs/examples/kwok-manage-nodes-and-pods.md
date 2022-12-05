# Manage nodes and pods with kwok

## Kwok with args `--manage-all-nodes=true`

Kwok will be in charge of all nodes in the cluster and maintain their heartbeats to API Server. In this way, all nodes will behave like real nodes and stay in `Ready` state.

## Kwok with args `--manage-nodes-with-annotation-selector=kwok.x-k8s.io/node=fake`

Kwok will be in charge of all Pods with annotation `kwok.x-k8s.io/node=fake`. If they carry an accurate `.spec.nodeName` filed, Kwok will ensure they stay in the `Running` state.

## Create a Node

With Kwok, you can join arbitrary Node(s) just by simply creating `v1.Node` object(s):

> The status can be any value.

``` bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Node
metadata:
  annotations:
    node.alpha.kubernetes.io/ttl: "0"
    kwok.x-k8s.io/node: fake
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/os: linux
    kubernetes.io/arch: amd64
    kubernetes.io/hostname: kwok-node-0
    kubernetes.io/os: linux
    kubernetes.io/role: agent
    node-role.kubernetes.io/agent: ""
    type: kwok
  name: kwok-node-0
spec:
  taints: # Avoid scheduling actual running pods to fake Node
    - effect: NoSchedule
      key: kwok.x-k8s.io/node
      value: fake
status:
  allocatable:
    cpu: 32
    memory: 256Gi
    pods: 110
  capacity:
    cpu: 32
    memory: 256Gi
    pods: 110
  nodeInfo:
    architecture: amd64
    bootID: ""
    containerRuntimeVersion: ""
    kernelVersion: ""
    kubeProxyVersion: fake
    kubeletVersion: fake
    machineID: ""
    operatingSystem: linux
    osImage: ""
    systemUUID: ""
  phase: Running
EOF
```

After the Node is created, Kwok will maintain its heartbeat and keep it in the `ready` state.

``` console
$ kubectl get node -o wide
NAME          STATUS   ROLES   AGE   VERSION   INTERNAL-IP   EXTERNAL-IP   OS-IMAGE    KERNEL-VERSION   CONTAINER-RUNTIME
kwok-node-0   Ready    agent   5s    fake      196.168.0.1   <none>        <unknown>   <unknown>        <unknown>
```

## Create a Pod

Now we create some Pods to verify if they can land on the previously created Nodes:

``` bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fake-pod
  namespace: default
spec:
  replicas: 10
  selector:
    matchLabels:
      app: fake-pod
  template:
    metadata:
      labels:
        app: fake-pod
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: type
                    operator: In
                    values:
                      - kwok
      # A taints was added to an automatically created Node.
      # You can remove taints of Node or add this tolerations.
      tolerations:
        - key: "kwok.x-k8s.io/node"
          operator: "Exists"
          effect: "NoSchedule"
      containers:
        - name: fake-container
          image: fake-image
EOF
```

After the Pod is created, we can see that all the Pods are landed on that Node(s) and are in the `Running` state.

``` console
$ kubectl get pod -o wide
NAME                        READY   STATUS    RESTARTS   AGE   IP          NODE          NOMINATED NODE   READINESS GATES
fake-pod-59bb47845f-4vl9f   1/1     Running   0          5s    10.0.0.5    kwok-node-0   <none>           <none>
fake-pod-59bb47845f-bc49m   1/1     Running   0          5s    10.0.0.4    kwok-node-0   <none>           <none>
fake-pod-59bb47845f-cnjsv   1/1     Running   0          5s    10.0.0.7    kwok-node-0   <none>           <none>
fake-pod-59bb47845f-g29wz   1/1     Running   0          5s    10.0.0.2    kwok-node-0   <none>           <none>
fake-pod-59bb47845f-gxq88   1/1     Running   0          5s    10.0.0.10   kwok-node-0   <none>           <none>
fake-pod-59bb47845f-pnzmn   1/1     Running   0          5s    10.0.0.9    kwok-node-0   <none>           <none>
fake-pod-59bb47845f-sfkk4   1/1     Running   0          5s    10.0.0.3    kwok-node-0   <none>           <none>
fake-pod-59bb47845f-vl2z5   1/1     Running   0          5s    10.0.0.8    kwok-node-0   <none>           <none>
fake-pod-59bb47845f-vpfhv   1/1     Running   0          5s    10.0.0.6    kwok-node-0   <none>           <none>
fake-pod-59bb47845f-wxn4b   1/1     Running   0          5s    10.0.0.1    kwok-node-0   <none>           <none>
```

## Update spec of nodes or pods

In a Kwok context, Nodes and Pods are nothing but pure API objects so feel free to mutate their API specs to do whatever simulation or testing you want.
