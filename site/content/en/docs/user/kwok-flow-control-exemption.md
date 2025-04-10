---
title: "Flow Control Exemption"
---

# Flow Control Exemption

{{< hint "warning" >}}
**Should only be used in non-production environments!**
{{< /hint >}}

The Kubernetes API server uses [flow control mechanisms][API Priority and Fairness] to prevent resource exhaustion, which can prevent components from making necessary API calls at scale.
To ensure reliable operation, you can exempt kwok from these rate limits using a FlowSchema resource.

## Configuration

``` yaml
apiVersion: flowcontrol.apiserver.k8s.io/v1
kind: FlowSchema
metadata:
  name: kwok-controller
spec:
  priorityLevelConfiguration:
    name: exempt # Bypass API server rate limiting 
  matchingPrecedence: 1000
  rules:
  - resourceRules:
    - apiGroups: ["*"]
      namespaces: ["*"]
      resources: ["*"]
      verbs: ["*"]
    subjects:
    - kind: ServiceAccount
      serviceAccount:
        name: kwok-controller
        namespace: kube-system
```

[API Priority and Fairness]: https://kubernetes.io/docs/concepts/cluster-administration/flow-control/
