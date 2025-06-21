package mixer

import (
	"fmt"
	"image"
	"log"
	"runtime"
	"strings"

	"github.com/fosdem/vidmix/ffmpegsource"
	"github.com/fosdem/vidmix/layer"
	"github.com/fosdem/vidmix/v4lsource"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

const windowWidth = 1280
const windowHeight = 720
const numLayers = 3
const f32 = 4

var (
	layers [3]*layer.Layer
)

var vertexShader = `
#version 400

in vec2 position;
in vec2 uv;

out vec2 UV;

void main() {
	gl_Position = vec4(position, 0.0, 1.0);
	UV = uv;
}
`

var fragmentShader = `
#version 400

in vec2 UV;

out vec4 color;

uniform sampler2D tex[9];
uniform vec4 sourcePosition[3];

vec4 sampleLayerYUV(int layer, vec4 dve) {
	vec2 tpos = (UV / dve.z) - (dve.xy / dve.z);
	float Y = texture(tex[layer*3], tpos).r;
	float Cb = texture(tex[layer*3+1], tpos).r - 0.5;
	float Cr = texture(tex[layer*3+2], tpos).r - 0.5;
	vec3 yuv = vec3(Y, Cr, Cb);
        mat3 colorMatrix = mat3(
                1,   0,       1.402,
                1,  -0.344,  -0.714,
                1,   1.772,   0);
	vec3 col = yuv * colorMatrix;
	float a = 1.0;
	if(tpos.x < 0 || tpos.x > 1.0) {
		a = 0.0;
	}
	if(tpos.y < 0 || tpos.y > 1.0) {
		a = 0.0;
	}
	return vec4(col.r, col.g, col.b, a);
}

vec4 sampleLayerRGB(int layer, vec4 dve) {
	return texture(tex[layer*3], (UV / dve.z) - (dve.xy / dve.z));
}

void main() {
	// vec4 background = sampleLayerRGB(0, sourcePosition[0]);
	vec4 background = sampleLayerYUV(0, sourcePosition[0]);
	vec4 dve1 = sampleLayerYUV(1, sourcePosition[1]);
	vec4 dve2 = sampleLayerYUV(2, sourcePosition[2]);
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

func MakeWindowAndMix() {
	window := makeWindow()
	initGL()

	program, err := newProgram(vertexShader, fragmentShader)
	if err != nil {
		log.Fatalf("Could not init shader: %s", err)
	}

	// layers[0] = layer.New(
	// 	"background",
	// 	imgsource.New("background.png"),
	// 	windowWidth, windowHeight,
	// )
	layers[0] = layer.New(
		"sauce",
		ffmpegsource.New(`
			ffmpeg -stream_loop -1 -re -i ~/s/random_shit/test_videos/cows.mp4 -vf scale=1920:1080 -pix_fmt yuyv422 -f rawvideo -r 60 -
			`,
		),
		windowWidth, windowHeight,
	)
	layers[1] = layer.New(
		"slides",
		v4lsource.New("/dev/video2", "yuyv", 1920, 1080),
		windowWidth, windowHeight,
	)
	layers[2] = layer.New(
		"cam",
		v4lsource.New("/dev/video0", "yuyv", 640, 480),
		windowWidth, windowHeight,
	)

	layers[0].Source.Start()
	layers[0].Move(0, 0, 1)

	layers[1].Source.Start()
	layers[1].Move(0.025, 0.049, 0.79)

	layers[2].Source.Start()
	layers[2].Move(0.75, 0.6, 0.2)

	for i := range layers {
		layers[i].SetupTextures()
	}

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

	var layerPos [numLayers * 4]float32
	layerPosUniform := gl.GetUniformLocation(program, gl.Str("sourcePosition\x00"))
	gl.Uniform4fv(layerPosUniform, numLayers, &layerPos[0])

	// Allocate 3 textures for every layer in case of planar YUV
	const numTextures = numLayers * 3
	var textures [numTextures]int32
	for i := range numTextures {
		textures[i] = int32(i)
	}
	texUniform := gl.GetUniformLocation(program, gl.Str("tex\x00"))
	gl.Uniform1iv(texUniform, numTextures, &textures[0])

	gl.ClearColor(1.0, 0.0, 0.0, 1.0)

	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Render
		gl.UseProgram(program)

		gl.BindVertexArray(vao)

		gl.Uniform1iv(texUniform, numTextures, &textures[0])
		for i := range numLayers {
			if !layers[i].Frames().IsReady {
				continue
			}
			rf := layers[i].Frames().LastFrame
			if rf == nil {
				continue
			}

			width := layers[i].Frames().Width
			height := layers[i].Frames().Height

			if layers[i].Frames().FrameType == layer.YUV422Frames {
				// Planar 4:2:2
				frm := rf.(*image.YCbCr)

				gl.ActiveTexture(uint32(gl.TEXTURE0 + (i * 3)))
				gl.BindTexture(gl.TEXTURE_2D, layers[i].TextureIDs[0])
				gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, int32(width), int32(height), gl.RED, gl.UNSIGNED_BYTE, gl.Ptr(frm.Y))

				gl.ActiveTexture(uint32(gl.TEXTURE0 + (i * 3) + 1))
				gl.BindTexture(gl.TEXTURE_2D, layers[i].TextureIDs[1])
				gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, int32(width/2), int32(height), gl.RED, gl.UNSIGNED_BYTE, gl.Ptr(frm.Cr))

				gl.ActiveTexture(uint32(gl.TEXTURE0 + (i * 3) + 2))
				gl.BindTexture(gl.TEXTURE_2D, layers[i].TextureIDs[2])
				gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, int32(width/2), int32(height), gl.RED, gl.UNSIGNED_BYTE, gl.Ptr(frm.Cb))

			} else {
				frm := rf.(*image.NRGBA)
				gl.ActiveTexture(uint32(gl.TEXTURE0 + (i * 3)))
				if !layers[i].Frames().IsStill {
					gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(frm.Pix))
				}
				gl.BindTexture(gl.TEXTURE_2D, layers[i].TextureIDs[0])
			}

			layerPos[(i*4)+0] = layers[i].Position.X
			layerPos[(i*4)+1] = layers[i].Position.Y
			layerPos[(i*4)+2] = layers[i].Size.X
			layerPos[(i*4)+3] = layers[i].Size.Y

			layers[i].Frames().RecycleFrame(rf)
		}
		gl.Uniform4fv(layerPosUniform, numLayers, &layerPos[0])

		gl.DrawArrays(gl.TRIANGLES, 0, 2*3)

		// Maintenance
		window.SwapBuffers()
		glfw.PollEvents()
	}
}
