{{- define "typeLink" -}}
{{- $groups := .groups -}}
{{- $separator := "" -}}
{{- if .multiline }}{{ $separator = "\n" }}{{ end -}}
{{- with .type -}}
{{- /* Kind: 3 = map, 4 = pointer, 5 = slice. */ -}}
{{- $leaf := . -}}
{{- range until 8 -}}
{{- if or (eq $leaf.Kind 4) (eq $leaf.Kind 5) -}}
{{- $leaf = $leaf.UnderlyingType -}}
{{- else if eq $leaf.Kind 3 -}}
{{- $leaf = $leaf.ValueType -}}
{{- end -}}
{{- end -}}
{{- $name := $leaf.Name -}}
{{- $link := "" -}}
{{- if index $groups $leaf.Package -}}
{{- $link = printf "#%s.%s" (index $groups $leaf.Package) $leaf.Name -}}
{{- else if regexMatch "^k8s\\.io/(api|apimachinery/pkg/apis)/" $leaf.Package -}}
{{- $parts := splitList "/" $leaf.Package -}}
{{- $version := last $parts -}}
{{- $group := index $parts (sub (len $parts) 2 | int) -}}
{{- $name = printf "Kubernetes %s/%s.%s" $group $version $leaf.Name -}}
{{- $link = printf "https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#%s-%s-%s" (lower $leaf.Name) $version $group -}}
{{- if and (eq $leaf.Name "Duration") (eq $leaf.Package "k8s.io/apimachinery/pkg/apis/meta/v1") -}}
{{- $link = "https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#Duration" -}}
{{- end -}}
{{- else if $leaf.Package -}}
{{- $name = printf "%s.%s" $leaf.Package $leaf.Name -}}
{{- end -}}
{{- if eq .Kind 3 -}}
{{- $name = .String -}}
{{- else if eq .Kind 5 -}}
{{- $name = printf "[]%s" $name -}}
{{- end -}}
{{- if $link }}<a href="{{ $link }}">{{ $separator }}{{ end -}}
{{- $name | html -}}
{{- if $link }}{{ $separator }}</a>{{ end -}}
{{- end -}}
{{- end -}}

{{- define "fields" -}}
{{- $groups := .groups }}
{{- range .type.Members }}
<tr>
<td>
<code>{{ .Name | html }}</code>
<em>
{{ template "typeLink" (dict "type" .Type "groups" $groups "multiline" true) }}
</em>
</td>
<td>
{{- if .Inlined }}
<p>(Members of <code>{{ .Name | html }}</code> are embedded into this type.)</p>
{{- end }}
{{- if index .Markers "optional" }}
<em>(Optional)</em>
{{- end }}
<p>{{ template "doc" .Doc }}</p>
{{- if eq .Type.Name "ObjectMeta" }}
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
{{- end }}
{{- if eq .Name "spec" }}
<table>
{{- template "fields" (dict "type" .Type "groups" $groups) }}
</table>
{{- end }}
</td>
</tr>
{{- end }}
{{- end -}}
