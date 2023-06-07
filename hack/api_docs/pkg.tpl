{{ define "packages" }}

---
title: API reference
bookToc: false
---

<h1>API reference</h1>

{{ with .packages }}
<p>Packages:</p>
<ul>
  {{ range . }}
  <li>
    <a href="#{{ packageAnchorID . }}">{{ packageDisplayName . }}</a>
  </li>
  {{ end }}
</ul>

{{ end }}

{{ range .packages }}
  <h2 id="{{ packageAnchorID . }}">
    {{ packageDisplayName . }}
    <a href="#{{ packageAnchorID . }}"> #</a>
  </h2>

  {{ with (index .GoPackages 0 ) }}
    {{ with .DocComments }}
    <div>
      {{ safe (renderComments .) }}
    </div>
    {{ end }}
  {{ end }}

  Resource Types:
  <ul>
  {{- range (visibleTypes (sortedTypes .Types)) -}}
    {{ if not (typeReferences .) }}
    <li>
      <a href="{{ linkForType . }}">{{ typeDisplayName . }}</a>
    </li>
    {{- end }}
  {{- end -}}
  </ul>

  {{ range (visibleTypes (sortedTypes .Types)) }}
    {{ if not (typeReferences .) }}
      {{ template "type" . }}
    {{ end }}
  {{ end }}
{{ end }}

<h2 id="references">
  References
  <a href="#references"> #</a>
</h2>

{{ range .packages }}
  {{ range (visibleTypes (sortedTypes .Types)) }}
    {{ if (typeReferences .) }}
      {{ template "type" . }}
    {{ end }}
  {{ end }}
{{ end }}

{{ end }}
