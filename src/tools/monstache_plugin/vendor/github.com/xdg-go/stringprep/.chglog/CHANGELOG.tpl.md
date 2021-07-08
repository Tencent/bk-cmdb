{{ if .Versions -}}
{{ range .Versions }}
<a name="{{ .Tag.Name }}"></a>
## [{{ .Tag.Name }}] - {{ datetime "2006-01-02" .Tag.Date }}
{{ range .CommitGroups }}
### {{ .Title }}
{{ range .Commits }}
- {{ if .Scope }}**{{ .Scope }}:** {{ end }}{{ .Subject }}
{{ end }}
{{ end -}}

{{- if .RevertCommits -}}
### Reverts
{{ range .RevertCommits -}}
- {{ .Revert.Header }}
{{ end }}
{{ end -}}

{{- if .NoteGroups -}}
{{ range .NoteGroups -}}
### {{ .Title }}
{{ range .Notes }}
{{ .Body }}
{{ end }}
{{ end -}}
{{ end -}}
{{ end }}
{{ range .Versions -}}
[{{ .Tag.Name }}]: {{ $.Info.RepositoryURL }}/releases/tag/{{ .Tag.Name }}
{{ end -}}
{{ end -}}
