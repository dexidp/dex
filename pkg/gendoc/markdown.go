package gendoc

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

var funcs = template.FuncMap{
	"renderJSON": func(i interface{}) string {
		if s, ok := i.(Schema); ok {
			return s.toJSON()
		}
		return ""
	},
	"toLink": toLink,
	"toCodeStr": func(code int) string {
		if code == CodeDefault {
			return "default"
		}
		return strconv.Itoa(code)
	},
}

var markdownTmpl = template.Must(template.New("md").Funcs(funcs).Parse(`
# {{ .Title }}

{{ .Description }}

__Version:__ {{ .Version }}

## Models

{{ range $i, $model := .Models }}
### {{ $model.Name }}

{{ $model.Description }}

{{ $model | renderJSON }}
{{ end }}

## Paths

{{ range $i, $path := .Paths }}
### {{ $path.Method }} {{ $path.Path }}

> __Summary__

> {{ $path.Summary }}

> __Description__

> {{ $path.Description }}

{{ if $path.Parameters }}
> __Parameters__

> |Name|Located in|Description|Required|Type|
|:-----|:-----|:-----|:-----|:-----|
{{ range $i, $p := $path.Parameters }}| {{ $p.Name }} | {{ $p.LocatedIn }} | {{ $p.Description }} | {{ if $p.Required }}Yes{{ else }}No{{ end }} | {{ $p.Type | toLink }} | 
{{ end }}
{{ end }}
> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
{{ range $i, $r := $path.Responses }}| {{ $r.Code | toCodeStr }} | {{ $r.Description }} | {{ $r.Type | toLink }} |
{{ end }}
{{ end }}
`))

func (doc Document) MarshalMarkdown() ([]byte, error) {
	var b bytes.Buffer
	if err := markdownTmpl.Execute(&b, doc); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (m Schema) toJSON() string {
	var b bytes.Buffer
	b.WriteString("```\n")
	m.writeJSON(&b, "", true)
	b.WriteString("\n```")
	return b.String()
}

var indentStr = "    "

func (m Schema) writeJSON(b *bytes.Buffer, indent string, first bool) {
	if m.Ref != "" {
		b.WriteString(m.Ref)
		return
	}
	if first {
		b.WriteString(indent)
	}

	switch m.Type {
	case TypeArray:
		b.WriteString("[")
		for i, c := range m.Children {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString("\n")
			b.WriteString(indent + indentStr)
			c.writeJSON(b, indent+indentStr, false)
		}
		b.WriteString("\n" + indent + "]")
	case TypeObject:
		b.WriteString("{")
		for i, c := range m.Children {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString("\n")
			b.WriteString(indent + indentStr + c.Name + ": ")
			c.writeJSON(b, indent+indentStr, false)
		}
		b.WriteString("\n" + indent + "}")
	case TypeBool, TypeFloat, TypeInt, TypeString:
		b.WriteString(m.Type)
		if m.Description != "" {
			b.WriteString(" // ")
			b.WriteString(m.Description)
		}
	}
}

func toAnchor(s string) string {
	var b bytes.Buffer
	r := strings.NewReader(s)
	for {
		r, _, err := r.ReadRune()
		if err != nil {
			return b.String()
		}
		switch {
		case r == ' ':
			b.WriteRune('-')
		case 'a' <= r && r <= 'z':
			b.WriteRune(r)
		case 'A' <= r && r <= 'Z':
			b.WriteRune(unicode.ToLower(r))
		}
	}
}

func toLink(s string) string {
	switch s {
	case "string", "boolean", "integer", "":
		return s
	}
	return fmt.Sprintf("[%s](#%s)", s, toAnchor(s))
}
