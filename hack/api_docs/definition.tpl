{{- define "doc" -}}
{{- $escaped := . | trim | html -}}
{{- $escaped = regexReplaceAll "`([^`]+)`" $escaped "<code>${1}</code>" -}}
{{- $escaped = replace "&#39;" "&rsquo;" $escaped -}}
{{- $escaped = regexReplaceAll "&#34;(.*?)&#34;" $escaped "&ldquo;${1}&rdquo;" -}}
{{- $escaped = replace "--" "&ndash;" $escaped -}}
{{- $escaped = regexReplaceAll "\n\n+" $escaped "</p>\n<p>" -}}
{{- regexReplaceAll "(https?://[^\\s<]*[^\\s<.,;:)])" $escaped "<a href=\"${1}\">${1}</a>" -}}
{{- end -}}

{{- define "definition" -}}
{{- $groups := .groups }}
{{- $group := .group }}
{{- with .type }}
<h3 id="{{ $group }}.{{ .Name }}">
{{ .Name }}
{{- if and .IsAlias .UnderlyingType.IsBasic }}
(<code>{{ .UnderlyingType.Name | html }}</code> alias)
{{- end }}
<a href="#{{ $group | replace "/" "%2f" }}.{{ .Name }}"> #</a>
</h3>
{{- with .SortedReferences }}
<p>
<em>Appears on: </em>
{{- range $index, $reference := . }}
{{- if $index }}{{ "\n, " }}{{ end }}
{{ template "typeLink" (dict "type" $reference "groups" $groups) }}
{{- end }}
</p>
{{- end }}
<p>
<p>{{ template "doc" .Doc }}</p>
</p>
{{- with .EnumValues }}
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody>
{{- $values := dict }}
{{- range . }}
{{- $_ := set $values .Name . }}
{{- end }}
{{- range keys $values | sortAlpha }}
{{- with index $values . }}
<tr>
<td><code>{{ printf "%q" .Name | html }}</code></td>
<td><p>{{ template "doc" .Doc }}</p>
</td>
</tr>
{{- end }}
{{- end }}
</tbody>
</table>
{{- end }}
{{- if .Members }}
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
{{- if not .References }}
<tr>
<td>
<code>apiVersion</code>
string
</td>
<td>
<code>
{{ $group }}
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code>
string
</td>
<td><code>{{ .Name }}</code></td>
</tr>
{{- end }}
{{- template "fields" (dict "type" . "groups" $groups) }}
</tbody>
</table>
{{- end }}
{{- end }}
{{- end -}}
