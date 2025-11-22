package shaders

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
)

var shaderCache map[string]uint32

func BuildGLProgram(shaderData *ShaderData) (uint32, error) {
	shaderer, err := NewShaderer()
	if err != nil {
		return 0, fmt.Errorf("could not get shaders: %w", err)
	}

	vertexShader, err := shaderer.GetShaderSource("screen.vert", shaderData)
	if err != nil {
		return 0, fmt.Errorf("could not get vertex shader: %w", err)
	}

	fragmentShader, err := shaderer.GetShaderSource("composite.frag", shaderData)
	if err != nil {
		return 0, fmt.Errorf("could not get vertex shader: %w", err)
	}

	writeFileDebug("/tmp/shader.vert", vertexShader)
	writeFileDebug("/tmp/shader.frag", fragmentShader)

	program, err := newProgram(vertexShader, fragmentShader)
	if err != nil {
		return 0, fmt.Errorf("could not init shader: %w", err)
	}

	return program, nil
}

func newProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	// FIXME: do we need this cache at all? isn't this called only once?
	// If we do, maybe we should put it into the shaderer

	vertexShader, ok := shaderCache[vertexShaderSource]
	if !ok {
		compiled, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
		if err != nil {
			return 0, err
		}
		vertexShader = compiled
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

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

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	size := int32(len(source))
	gl.ShaderSource(shader, 1, csources, &size)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		clog := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(clog))

		return 0, fmt.Errorf("failed to compile %v: %v", source, clog)
	}

	return shader, nil
}

func writeFileDebug(filename string, content string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("could not create debug file %s: %s", filename, err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			return
		}
	}(f)

	_, err = fmt.Fprintf(f, "%s", content)
	if err != nil {
		log.Printf("Could not write to debug file: %s", err)
		return
	}
}
