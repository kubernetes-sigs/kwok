apiVersion: v1
kind: Config
preferences: {}
current-context: {{ .ProjectName }}
clusters:
  - name: {{ .ProjectName }}
    cluster:
      server: {{ .Address }}
{{ if .SecurePort }}
      insecure-skip-tls-verify: true
{{ end}}
contexts:
  - name: {{ .ProjectName }}
    context:
      cluster: {{ .ProjectName }}

{{ if .SecurePort }}
      user: {{ .ProjectName }}
users:
  - name: {{ .ProjectName }}
    user:
      client-certificate: {{ .AdminCrtPath }}
      client-key: {{ .AdminKeyPath }}
{{ end}}
