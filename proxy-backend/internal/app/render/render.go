package render

import (
	"embed"
	"html/template"
	"strings"
)

//go:embed templates
var templates embed.FS

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if strings.EqualFold(strings.TrimSpace(v), item) {
			return true
		}
	}
	return false
}

func ParseHtmlTemplates() (*template.Template, error) {
	tmpl := template.New("").Funcs(template.FuncMap{"contains": contains})
	tmpl, err := tmpl.ParseFS(templates, "templates/*go.tmpl")
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}
