{{ define "type" }}

<h3 id="{{ anchorIDForType . }}">
  {{ .Name.Name }}
  {{ if eq .Kind "Alias" }}(<code>{{ .Underlying }}</code> alias){{ end }}
  <a href="#{{ anchorIDForType . }}"> #</a>
</h3>
{{ with (typeReferences .) }}
  <p>
    <em>Appears on: </em>
    {{ $prev := "" }}
    {{ range . }}
      {{ if $prev }}, {{ end }}
      {{ $prev = . }}
      <a href="{{ linkForType . }}">{{ typeDisplayName . }}</a>
    {{ end }}
  </p>
{{ end }}

<p>
  {{ safe (renderComments .CommentLines) }}
</p>

{{ with (constantsOfType .) }}
<table>
  <thead>
    <tr>
      <th>Value</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    {{ range . }}
    <tr>
      <td><code>{{ typeDisplayName . }}</code></td>
      <td>{{ safe (renderComments .CommentLines) }}</td>
    </tr>
    {{ end }}
  </tbody>
</table>
{{ end }}

{{ if .Members }}
<table>
  <thead>
    <tr>
      <th>Field</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    {{ if not (typeReferences .) }}
    <tr>
      <td>
        <code>apiVersion</code>
        string
      </td>
      <td>
        <code>
          {{ apiGroup . }}
        </code>
      </td>
    </tr>
    <tr>
      <td>
        <code>kind</code>
        string
      </td>
      <td><code>{{ .Name.Name }}</code></td>
    </tr>
    {{ end }}
    {{ template "members" . }}
  </tbody>
</table>
{{ end }}

{{ end }}
