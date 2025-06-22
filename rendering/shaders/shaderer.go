package shaders

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"github.com/fosdem/fazantix/theatre"
)

//go:embed *.frag *.vert
var templateDir embed.FS

type Shaderer struct {
	templates *template.Template
	theatre   *theatre.Theatre

	VertexShader   uint32
	FragmentShader uint32
}

func NewShaderer(theatre *theatre.Theatre) (*Shaderer, error) {
	s := &Shaderer{}

	var err error

	s.theatre = theatre
	s.templates, err = template.ParseFS(templateDir, "*.frag", "*.vert")

	return s, err
}

func (s *Shaderer) GetShaderSource(name string) (string, error) {
	var b bytes.Buffer
	err := s.templates.ExecuteTemplate(&b, name, s.theatre)
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
