{{- define "gvList" -}}
---
title: API reference
bookToc: false
---
<h1>API reference</h1>
<p>Packages:</p>
<ul>
{{- $groups := dict }}
{{- range . }}
{{- $group := .GroupVersionString }}
{{- range .SortedTypes }}
{{- $_ := set $groups .Package $group }}
{{- end }}
<li>
<a href="#{{ $group | replace "/" "%2f" }}">{{ $group }}</a>
</li>
{{- end }}
</ul>
{{- range . }}
{{- $group := .GroupVersionString }}
<h2 id="{{ $group }}">
{{ $group }}
<a href="#{{ $group | replace "/" "%2f" }}"> #</a>
</h2>
<div>
<p>{{ template "doc" .Doc }}</p>
</div>
Resource Types:
<ul>
{{- range .SortedTypes }}
{{- if not .References }}
<li>
<a href="#{{ $group }}.{{ .Name }}">{{ .Name }}</a>
</li>
{{- end }}
{{- end }}</ul>
{{- range .SortedTypes }}
{{- if not .References }}
{{- template "definition" (dict "type" . "group" $group "groups" $groups) }}
{{- end }}
{{- end }}
{{- end }}
<h2 id="references">
References
<a href="#references"> #</a>
</h2>
{{- range . }}
{{- $group := .GroupVersionString }}
{{- range .SortedTypes }}
{{- if .References }}
{{- template "definition" (dict "type" . "group" $group "groups" $groups) }}
{{- end }}
{{- end }}
{{- end }}
{{ end -}}
