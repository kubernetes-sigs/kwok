---
title: "Ray"
---

# Ray

More information about Ray can be found [here](https://docs.ray.io/en/latest/cluster/kubernetes/index.html).

## Set up Cluster

Create a new KWOK cluster using the kind runtime
``` bash
kwokctl create cluster --runtime kind
```

## Create Node

Create a KWOK fake node
``` bash
kubectl apply -f https://kwok.sigs.k8s.io/examples/node.yaml
```

Verify that the nodes are created and running
```bash
kubectl get node
kNAME                      STATUS                     ROLES           AGE     VERSION
kwok-kwok-control-plane   Ready,SchedulingDisabled   control-plane   3m33s   v1.33.0
kwok-node-0               Ready                      agent           3m11s   kwok-v0.7.0
```

## Deploy Ray Operator

Add the KubeRay Helm repository and install the KubeRay operator using Helm
```bash
helm repo add kuberay https://ray-project.github.io/kuberay-helm/
helm install kuberay-operator kuberay/kuberay-operator --version 1.4.2
```

Patch the KubeRay operator deployment to run on the control plane node
```bash
kubectl patch deploy kuberay-operator --type=json -p='[{"op":"add","path":"/spec/template/spec/nodeName","value":"kwok-kwok-control-plane"}]'
```

## Create Mock Ray Head

Create a ConfigMap containing a mock Ray head server script
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: mock-head
  namespace: default
data:
  server.py: |
    from fastapi import FastAPI
    import uvicorn

    app = FastAPI()

    @app.get("/api/jobs/{job_id}")
    def get_job_info(job_id: str):
        return {
            "job_id": job_id,
            "status": "SUCCEEDED",
        }

    if __name__ == "__main__":
        uvicorn.run(app, host="0.0.0.0", port=8265)
EOF
```

Create a Deployment for the mock Ray head service
```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mock-head
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mock-head
  template:
    metadata:
      labels:
        app: mock-head
    spec:
      nodeName: kwok-kwok-control-plane
      containers:
      - name: server
        image: tiangolo/uvicorn-gunicorn-fastapi:python3.11
        ports:
        - containerPort: 8265
        volumeMounts:
        - name: mock-head
          mountPath: /app
        command: ["python"]
        args: ["/app/server.py"]
        env:
        - name: PORT
          value: "8265"
      volumes:
      - name: mock-head
        configMap:
          name: mock-head
EOF
```

Create a Service to expose the mock Ray head
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: mock-head
  namespace: default
spec:
  ports:
  - port: 8265
    targetPort: 8265
    protocol: TCP
  selector:
    app: mock-head
EOF
```

## Update CoreDNS

Get the current CoreDNS configuration and save it to a file
```bash
kubectl get configmap coredns -n kube-system -o yaml > coredns.yaml
```

Add a rewrite rule to redirect Ray head service DNS queries to the mock service
```bash
sed -i '' '/ready/a\
        rewrite name regex (.+)-head-svc\.(.+)\.svc\.cluster\.local mock-head.default.svc.cluster.local
' coredns.yaml
```

Apply the updated CoreDNS configuration
```bash
kubectl apply -f coredns.yaml
```

Restart the CoreDNS deployment to run on the control plane node to reload the configuration
```bash
kubectl patch deployment coredns -n kube-system --type=json -p='[{"op":"add","path":"/spec/template/spec/nodeName","value":"kwok-kwok-control-plane"}]'
```

## Test RayJob

Deploy a sample Ray job to test the setup
``` bash
kubectl apply -f https://raw.githubusercontent.com/ray-project/kuberay/master/ray-operator/config/samples/pytorch-mnist/ray-job.pytorch-mnist.yaml
```

Check the status of the Ray job
```bash
kubectl get rayjob
NAME                   JOB STATUS   DEPLOYMENT STATUS   START TIME             END TIME               AGE
rayjob-pytorch-mnist   SUCCEEDED    Complete            2025-08-16T18:52:01Z   2025-08-16T18:52:02Z   5m31s
```
