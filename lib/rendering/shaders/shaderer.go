package shaders

import (
	"bytes"
	"embed"
	"fmt"
	"image/color"
	"text/template"

	"github.com/fosdem/fazantix/lib/layer"
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

	var err error

	s.templates, err = template.ParseFS(templateDir, "*.frag", "*.vert")

	return s, err
}

// ShaderData contains stuff that gets passed to the shader
type ShaderData struct {
	Sources        []layer.Source
	NumSources     uint32
	NumLayers      uint32
	FallbackColour color.RGBA
}

func (s *Shaderer) GetShaderSource(name string, data *ShaderData) (string, error) {
	var b bytes.Buffer
	err := s.templates.ExecuteTemplate(&b, name, data)
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
