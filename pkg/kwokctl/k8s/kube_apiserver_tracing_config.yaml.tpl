apiVersion: apiserver.config.k8s.io/v1alpha1
kind: TracingConfiguration
endpoint: {{ .Endpoint }}
samplingRatePerMillion: 1000000
