package shaders

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

//go:embed *.frag *.vert
var templateDir embed.FS

type Shaderer struct {
	templates *template.Template

	VertexShader   uint32
	FragmentShader uint32
	Program        uint32
}

func BuildShaderer() (*Shaderer, error) {
	s, err := newShaderer()
	if err != nil {
		return nil, err
	}

	s.Program, err = s.newProgram()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func newShaderer() (*Shaderer, error) {
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

func (s *Shaderer) compileShader(name string, shaderType uint32) (uint32, error) {
	source, err := s.GetShaderSource("screen.vert")
	if err != nil {
		return 0, err
	}

	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", name, log)
	}

	return shader, nil
}

func (s *Shaderer) newProgram() (uint32, error) {
	vertexShader, err := s.compileShader("screen.vert", gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := s.compileShader("composite.frag", gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	s.VertexShader = vertexShader
	s.FragmentShader = fragmentShader

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)
	glfw.SwapInterval(1)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		logmsg := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(logmsg))

		return 0, fmt.Errorf("failed to link program: %v", logmsg)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program, nil
}
