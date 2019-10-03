package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

const rawTemplate = `// Code generated with {{ .Command }}; DO NOT EDIT.
package {{ .PackageName }}

func init() {
	{{ .Name }} = {{ printf "%#v" .Value }}
}
`

type templateData struct {
	PackageName string
	Command     string
	Name        string
	Value       []byte
}

func main() {
	var (
		pkg  = flag.String("package", "main", "output file package name")
		out  = flag.String("out", "packed.go", "output file name")
		in   = flag.String("in", "in", "input file name")
		name = flag.String("name", "in", "variable name to set")
	)

	flag.Parse()

	value, err := ioutil.ReadFile(*in)
	if err != nil {
		panic(fmt.Errorf("reading input file: %w", err))
	}

	tmpl, _ := template.New("pack").Parse(rawTemplate)

	outFile, err := os.Create(*out)
	if err != nil {
		panic(fmt.Errorf("creating output file: %w", err))
	}

	if err := tmpl.Execute(outFile, templateData{
		PackageName: *pkg,
		Command:     strings.Join(os.Args, " "),
		Name:        *name,
		Value:       value,
	}); err != nil {
		panic(fmt.Errorf("executing template: %w", err))
	}
}
