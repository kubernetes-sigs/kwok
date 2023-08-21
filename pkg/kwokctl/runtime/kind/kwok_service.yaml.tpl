apiVersion: v1
kind: Service
metadata:
  labels:
    k8s-app: kwok-controller
  name: kwok-controller
  namespace: kube-system
spec:
  ports:
  - name: http
    port: 10247
    protocol: TCP
    targetPort: http
  selector:
    k8s-app: kwok-controller
