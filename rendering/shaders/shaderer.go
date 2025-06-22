package shaders

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed *.frag *.vert
var templateDir embed.FS

type Shaderer struct {
	templates *template.Template

	VertexShader   uint32
	FragmentShader uint32
}

func NewShaderer() (*Shaderer, error) {
	s := &Shaderer{}

	s.templates = template.New("root")
	s.templates.ParseFS(templateDir, "*.frag", "*.vert")

	return s, nil
}

func (s *Shaderer) GetShaderSource(name string) (string, error) {
	var b bytes.Buffer
	err := s.templates.ExecuteTemplate(&b, name, "no data yet")
	if err != nil {
		return "", fmt.Errorf("error while rendering template: %s", err)
	}

	return b.String(), nil
}

func (s *Shaderer) TemplateNames() []string {
	var names []string
	for _, t := range s.templates.Templates() {
		names = append(names, t.Name())
	}
	return names
}
