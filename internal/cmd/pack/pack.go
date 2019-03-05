package main

import (
	"flag"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/pkg/errors"
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
		panic(errors.Wrap(err, "reading input file"))
	}

	tmpl, _ := template.New("pack").Parse(rawTemplate)

	outFile, err := os.Create(*out)
	if err != nil {
		panic(errors.Wrap(err, "creating output file"))
	}

	if err := tmpl.Execute(outFile, templateData{
		PackageName: *pkg,
		Command:     strings.Join(os.Args, " "),
		Name:        *name,
		Value:       value,
	}); err != nil {
		panic(errors.Wrap(err, "executing template"))
	}
}
