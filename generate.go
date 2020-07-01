package main

import (
	"bytes"
	"go/format"
	"go/token"
	"os"
	"strings"
	"text/template"
)

const variantTemplate = `{{- $v := . -}}
// Code generated by {{command}}; DO NOT EDIT.
package {{.Package}}

import ({{range .Imports}}  
	{{.}} 
{{- end}}
)

type {{.Name}} {{gofmt .Methods}}
{{if .Visitor}} 
type {{.Name}}Visitor struct {
{{- range .Constructors}}
	{{title .Name}} func{{.UnpackTypes false}}
{{- end}}
} 
{{end -}}
{{range .Constructors}}
type {{.Name}} {{gofmt .Type}}
{{end -}}
{{range .Constructors}}
func ({{.Name}}) is{{title $v.Name}}() {}
{{if $v.Unpack -}} func (rcv {{.Name}}) Unpack() {{.UnpackTypes true}} { return {{.UnpackValues}} }
{{end -}}
{{if $v.Visitor -}} func (rcv {{.Name}}) Visit(v {{$v.Name}}Visitor) { v.{{title .Name}}({{.UnpackValues}}) } 
{{end}}{{end}}

{{- if $v.ExplicitCheck}}
var ({{range .Constructors}}
	_ {{$v.Name}} = struct{ {{.Name}} }{}
{{- end}}
)
{{end}}`

// gofmt formats source code using gofmt as a library
func gofmt(node interface{}) (string, error) {

	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), node); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// command returns the commandline that generated the file
func command() string {
	args := os.Args
	args[0] = "govariant"
	return strings.Join(args, " ")
}

func generateSourceCode(variantStruct variant) ([]byte, error) {
	// create template instance
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"gofmt":   gofmt,
		"command": command,
		"title":   strings.Title,
	}).Parse(variantTemplate)
	if err != nil {
		return nil, err
	}
	// execute template, creating types and methods
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variantStruct); err != nil {
		return nil, err
	}

	// format generated source code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, err
	}
	return formatted, nil
}