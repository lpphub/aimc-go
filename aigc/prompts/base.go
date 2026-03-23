package prompts

import (
	"bytes"
	"text/template"
)

// NewPrompt renders a template string with named variables.
// Template syntax: "Hello {{.Name}}, welcome to {{.Place}}"
func NewPrompt(tmpl string, vars map[string]any) (string, error) {
	t, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}
