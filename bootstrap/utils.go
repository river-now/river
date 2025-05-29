package bootstrap

import (
	"os"
	"strings"
	"text/template"
)

func (d *derivedOptions) tmplWriteMust(target, tmplStr string) {
	tmpl := template.Must(template.New(target).Parse(tmplStr))
	var sb strings.Builder
	if err := tmpl.Execute(&sb, d); err != nil {
		panic(err)
	}
	b := []byte(sb.String())
	if err := os.WriteFile(target, b, 0644); err != nil {
		panic(err)
	}
}

func strWriteMust(target string, content string) {
	if err := os.WriteFile(target, []byte(content), 0644); err != nil {
		panic(err)
	}
}
