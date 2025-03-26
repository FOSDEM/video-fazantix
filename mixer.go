package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"


	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
//	"github.com/go-gl/mathgl/mgl32"
)

const windowWidth = 1280
const windowHeight = 720
const f32 = 4

var (
	sources [3]Source
)

var vertexShader = `
#version 330

in vec2 position;
in vec2 uv;

out vec2 UV;

void main() {
	gl_Position = vec4(position, 0.0, 1.0);
	UV = uv;
}
`

var fragmentShader = `
#version 330

in vec2 UV;

out vec4 color;

uniform sampler2D tex[3];
uniform vec4 sourcePosition[3];

void main() {
	vec4 background = texture(tex[0], UV + vec2(0.5, 0.5) - sourcePosition[0].xy);
	
	vec4 dve1 = texture(tex[1], (UV / sourcePosition[1].z) - (sourcePosition[1].xy / sourcePosition[1].z));
	vec4 dve2 = texture(tex[2], (UV / sourcePosition[2].zw) - (sourcePosition[2].xy / sourcePosition[2].zw));
	vec4 temp = mix(background, dve1, dve1.a);
	color = mix(temp, dve2, dve2.a);

}
`

var vertices = []float32{
	//  X, Y,  U, V
	-1.0, -1.0, 0.0, 1.0,
	+1.0, -1.0, 1.0, 1.0,
	+1.0, +1.0, 1.0, 0.0,

	-1.0, -1.0, 0.0, 1.0,
	+1.0, +1.0, 1.0, 0.0,
	-1.0, +1.0, 0.0, 0.0,
}

func init() {
	// The OpenGL stuff must be in one thread
	runtime.LockOSThread()
}

func makeWindow() *glfw.Window {
	log.Println("Initializing window")
	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	//defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	window, err := glfw.CreateWindow(windowWidth, windowHeight, "OpenGL", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	return window
}

func initGL() {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Printf("OpenGL version '%s'", version)	
}

func compileShader(source string, shaderType uint32) (uint32, error) {
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

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

func newProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
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

func main() {
	window := makeWindow()
	initGL()

	program, err := newProgram(vertexShader, fragmentShader)
	if err != nil {
		log.Fatalf("Could not init shader: %s", err)
	}

	sources[0] = newSource("background")
	sources[1] = newSource("slides")
	sources[2] = newSource("cam")

	sources[0].LoadStill("background.png")
	sources[1].LoadStill("slides.png")
	sources[1].Move(0.025, 0.049, 0.79)

	sources[2].LoadV4l2("/dev/video0", 640, 480)
	sources[2].Move(0.75, 0.6, 0.2)


	// Configure the vertex data
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*f32, gl.Ptr(vertices), gl.STATIC_DRAW)
	
	var stride int32
	stride = 4 * f32

	vertAttrib := uint32(gl.GetAttribLocation(program, gl.Str("position\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointerWithOffset(vertAttrib, 2, gl.FLOAT, false, stride, 0)

	texCoordAttrib := uint32(gl.GetAttribLocation(program, gl.Str("uv\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointerWithOffset(texCoordAttrib, 2, gl.FLOAT, false, stride, 2*f32)

	var layerPos [3*4]float32
	layerPosUniform := gl.GetUniformLocation(program, gl.Str("sourcePosition\x00"))
	gl.Uniform4fv(layerPosUniform, 3, &layerPos[0])

	var textures [3]int32
	textures[0] = 0
	textures[1] = 1
	textures[2] = 2
	texUniform := gl.GetUniformLocation(program, gl.Str("tex\x00"))
	gl.Uniform1iv(texUniform, 3, &textures[0])

	gl.ClearColor(1.0, 0.0, 0.0, 1.0)

	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Render
		gl.UseProgram(program)

		gl.BindVertexArray(vao)

		

		gl.Uniform1iv(texUniform, 3, &textures[0])
		for i := range 3 {
			if sources[i].IsReady {

				gl.ActiveTexture(uint32(gl.TEXTURE0 + i))
				if !sources[i].IsStill {
					frm := <-sources[i].Images
					gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, 640, 480, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(frm.Pix))
				}
				gl.BindTexture(gl.TEXTURE_2D, sources[i].Texture)

				layerPos[(i*4)+0] = sources[i].Position.x
				layerPos[(i*4)+1] = sources[i].Position.y
				layerPos[(i*4)+2] = sources[i].Size.x
				layerPos[(i*4)+3] = sources[i].Size.y
			}
		}
		gl.Uniform4fv(layerPosUniform, 3, &layerPos[0])

		gl.DrawArrays(gl.TRIANGLES, 0, 2*3)

		// Maintenance
		window.SwapBuffers()
		glfw.PollEvents()
	}
}
