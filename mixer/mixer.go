package mixer

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/fosdem/fazantix/api"
	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/ffmpegsource"
	"github.com/fosdem/fazantix/imgsource"
	"github.com/fosdem/fazantix/layer"
	"github.com/fosdem/fazantix/rendering"
	"github.com/fosdem/fazantix/rendering/shaders"
	"github.com/fosdem/fazantix/theatre"
	"github.com/fosdem/fazantix/v4lsource"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

const f32 = 4

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

func makeWindow(cfg *config.WindowCfg) *glfw.Window {
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
	window, err := glfw.CreateWindow(cfg.W, cfg.H, "OpenGL", nil, nil)
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

func writeFileDebug(filename string, content string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("could not create debug file %s: %s", filename, err)
	}
	defer f.Close()

	fmt.Fprintf(f, "%s", content)
}

func MakeWindowAndMix(cfg *config.Config) {
	window := makeWindow(cfg.Window)
	initGL()

	theatre := makeTheatre(cfg)
	layers := theatre.Layers

	var theApi *api.Api
	if cfg.Api != nil {
		theApi = api.New(cfg.Api, theatre)

		log.Printf("starting web server\n")
		go func() {
			err := theApi.Serve()
			if err != nil {
				log.Fatalf("could not start web server: %s", err)
			}
		}()
	}

	numLayers := int32(len(layers))

	shaderer, err := shaders.NewShaderer(theatre)
	if err != nil {
		log.Fatalf("Could not get shaders: %s", err)
	}

	vertexShader, err := shaderer.GetShaderSource("screen.vert")
	if err != nil {
		log.Fatalf("Could not get vertex shader: %s", err)
	}

	fragmentShader, err := shaderer.GetShaderSource("composite.frag")
	if err != nil {
		log.Fatalf("Could not get vertex shader: %s", err)
	}

	writeFileDebug("/tmp/shader.vert", vertexShader)
	writeFileDebug("/tmp/shader.frag", fragmentShader)

	program, err := newProgram(vertexShader, fragmentShader)
	if err != nil {
		log.Fatalf("Could not init shader: %s", err)
	}

	theatre.Start()
	err = theatre.SetScene("default")
	if err != nil {
		log.Fatalf("Could not apply default scene")
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

	layerPos := make([]float32, numLayers*4)
	layerPosUniform := gl.GetUniformLocation(program, gl.Str("layerPosition\x00"))
	gl.Uniform4fv(layerPosUniform, numLayers, &layerPos[0])

	layerData := make([]float32, numLayers*4)
	layerDataUniform := gl.GetUniformLocation(program, gl.Str("layerData\x00"))
	gl.Uniform4fv(layerDataUniform, numLayers, &layerData[0])

	// Allocate 3 textures for every layer in case of planar YUV
	numTextures := numLayers * 3
	textures := make([]int32, numTextures)
	for i := range numTextures {
		textures[i] = int32(i)
	}
	texUniform := gl.GetUniformLocation(program, gl.Str("tex\x00"))
	gl.Uniform1iv(texUniform, numTextures, &textures[0])

	gl.ClearColor(1.0, 0.0, 0.0, 1.0)

	firstFrame := true
	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Render
		gl.UseProgram(program)

		gl.BindVertexArray(vao)

		gl.Uniform1iv(texUniform, numTextures, &textures[0])
		for i := range numLayers {
			if layers[i].Frames().IsStill && !firstFrame {
				continue
			}
			if !layers[i].Frames().IsReady {
				continue
			}
			rf := layers[i].Frames().LastFrame
			if rf == nil {
				continue
			}

			rendering.SendFrameToGPU(rf, layers[i].TextureIDs, int(i))

			layerPos[(i*4)+0] = layers[i].Position.X
			layerPos[(i*4)+1] = layers[i].Position.Y
			layerPos[(i*4)+2] = layers[i].Size.X
			layerPos[(i*4)+3] = layers[i].Size.Y
			layerData[(i*4)+0] = layers[i].Opacity
			layerData[(i*4)+1] = layers[i].Border

			layers[i].Frames().RecycleFrame(rf)
		}
		theatre.Animate()
		gl.Uniform4fv(layerDataUniform, numLayers, &layerData[0])
		gl.Uniform4fv(layerPosUniform, numLayers, &layerPos[0])

		gl.DrawArrays(gl.TRIANGLES, 0, 2*3)

		// Maintenance
		window.SwapBuffers()
		glfw.PollEvents()
		firstFrame = false

		if theApi != nil {
			theApi.FrameCounter++
		}
	}
}

func makeTheatre(cfg *config.Config) *theatre.Theatre {
	enabledSources := make(map[string]struct{})
	for _, layerStateMap := range cfg.Scenes {
		for name := range layerStateMap {
			if _, ok := cfg.Sources[name]; ok {
				enabledSources[name] = struct{}{}
			} else {
				log.Fatalf("no such source: %s", name)
			}
		}
	}

	var sortedSourceNames []string
	for name := range enabledSources {
		sortedSourceNames = append(sortedSourceNames, name)
	}

	sort.Slice(sortedSourceNames, func(i, j int) bool {
		ni := sortedSourceNames[i]
		nj := sortedSourceNames[j]
		return cfg.Sources[ni].Z < cfg.Sources[nj].Z
	})

	var sources []layer.Source
	for _, srcName := range sortedSourceNames {
		srcCfg := cfg.Sources[srcName]

		log.Printf("adding source: %s\n", srcName)

		switch sc := srcCfg.Cfg.(type) {
		case *config.FFmpegSourceCfg:
			sources = append(sources, ffmpegsource.New(srcName, sc))
		case *config.ImgSourceCfg:
			sources = append(sources, imgsource.New(srcName, sc))
		case *config.V4LSourceCfg:
			sources = append(sources, v4lsource.New(srcName, sc))
		default:
			panic(fmt.Sprintf("unhandled source type: %+v", srcCfg.Cfg))
		}
	}

	scenes := make(map[string]*theatre.Scene)
	for sceneName, layerStateMap := range cfg.Scenes {
		layerStates := make([]*layer.LayerState, len(sources))
		for i, src := range sources {
			layerStates[i] = layerStateMap[src.Name()]
		}
		scenes[sceneName] = &theatre.Scene{
			Name:        sceneName,
			LayerStates: layerStates,
		}
	}
	return theatre.New(sources, scenes, cfg.Window.W, cfg.Window.H)
}
