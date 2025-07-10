package mixer

import (
	"fmt"
	"log"
	"maps"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/fosdem/fazantix/api"
	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/rendering"
	"github.com/fosdem/fazantix/rendering/shaders"
	"github.com/fosdem/fazantix/theatre"
	"github.com/fosdem/fazantix/windowsink"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

const f32 = 4

var currentTheatre *theatre.Theatre

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

func keyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Release {
		if key == glfw.KeyQ &&
			mods&glfw.ModControl != 0 &&
			mods&glfw.ModShift != 0 {
			log.Println("told to quit, exiting")
			w.SetShouldClose(true)
		}
	}
	if action == glfw.Press {
		if key >= glfw.Key0 && key <= glfw.Key9 {
			selected := int(key - glfw.Key0)
			if selected > len(currentTheatre.Scenes)-1 {
				log.Printf("Scene %d out of range\n", selected)
				return
			}
			log.Println("Scene ", selected)
			scenes := slices.Sorted(maps.Keys(currentTheatre.Scenes))
			names := make([]string, len(currentTheatre.Scenes))
			for i, n := range scenes {
				names[i] = n
			}
			log.Printf("set scene %s\n", names[selected])
			err := currentTheatre.SetScene("projector", names[selected])
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func makeWindow(sink *windowsink.WindowSink) *glfw.Window {
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
	window, err := glfw.CreateWindow(sink.Frames().Width, sink.Frames().Height, sink.Frames().Name, nil, nil)
	if err != nil {
		panic(err)
	}

	window.SetKeyCallback(keyCallback)
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

var shaderCache map[string]uint32

func newProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
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

func MakeWindowAndMix(cfg *config.Config) {
	alloc := &encdec.DumbFrameAllocator{}
	var err error
	currentTheatre, err = theatre.New(cfg, alloc)
	if err != nil {
		log.Fatalf("could not build theatre: %s", err)
	}

	// assume a single sink called "projector" of type "window" for now
	windowStage := currentTheatre.GetTheSingleWindowStage()
	window := makeWindow(windowStage.Sink.(*windowsink.WindowSink))
	initGL()

	layers := windowStage.Layers

	var theApi *api.Api
	if cfg.Api != nil {
		theApi = api.New(cfg.Api, currentTheatre)

		log.Printf("starting web server\n")
		go func() {
			err := theApi.Serve()
			if err != nil {
				log.Fatalf("could not start web server: %s", err)
			}
		}()
	}

	shaderData := &shaders.ShaderData{
		NumSources: currentTheatre.NumSources(),
		Sources:    currentTheatre.SourceList,
	}

	numLayers := int32(len(layers))

	shaderer, err := shaders.NewShaderer()
	if err != nil {
		log.Fatalf("Could not get shaders: %s", err)
	}

	vertexShader, err := shaderer.GetShaderSource("screen.vert", shaderData)
	if err != nil {
		log.Fatalf("Could not get vertex shader: %s", err)
	}

	fragmentShader, err := shaderer.GetShaderSource("composite.frag", shaderData)
	if err != nil {
		log.Fatalf("Could not get vertex shader: %s", err)
	}

	writeFileDebug("/tmp/shader.vert", vertexShader)
	writeFileDebug("/tmp/shader.frag", fragmentShader)

	program, err := newProgram(vertexShader, fragmentShader)
	if err != nil {
		log.Fatalf("Could not init shader: %s", err)
	}

	currentTheatre.Start()
	err = currentTheatre.SetScene("projector", "default")
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

	stride := int32(4 * f32)

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

	stageDataUniform := gl.GetUniformLocation(program, gl.Str("stageData\x00"))
	gl.Uniform1ui(stageDataUniform, 0)

	// Allocate 3 textures for every layer in case of planar YUV
	numTextures := numLayers * 3
	textures := make([]int32, numTextures)
	for i := range numTextures {
		textures[i] = int32(i)
	}
	texUniform := gl.GetUniformLocation(program, gl.Str("tex\x00"))
	gl.Uniform1iv(texUniform, numTextures, &textures[0])

	// Create extra framebuffers as rendertargets
	nonWindowStages := currentTheatre.NonWindowStageList

	for _, stage := range nonWindowStages {
		stage.Sink.Frames().SetupTextures()
		stage.Sink.Frames().UseAsFramebuffer()
	}

	for name := range currentTheatre.Stages {
		err := currentTheatre.SetScene(name, "default")
		if err != nil {
			log.Fatalf("Could not apply default scene: %s", err)
		}
	}

	gl.ClearColor(1.0, 0.0, 0.0, 1.0)

	frameCounter := 0
	frameTimer := time.Now()
	deltaTimer := time.Now()
	firstFrame := true
	for !window.ShouldClose() {
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		gl.Viewport(0, 0, int32(windowStage.Sink.Frames().Width), int32(windowStage.Sink.Frames().Height))
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Render
		gl.UseProgram(program)

		gl.BindVertexArray(vao)

		gl.Uniform1iv(texUniform, numTextures, &textures[0])
		gl.Uniform1ui(stageDataUniform, windowStage.StageData())
		layers = windowStage.Layers
		for i := range numLayers {
			layerPos[(i*4)+0] = layers[i].Position.X
			layerPos[(i*4)+1] = layers[i].Position.Y
			layerPos[(i*4)+2] = layers[i].Size.X
			layerPos[(i*4)+3] = layers[i].Size.Y
			layerData[(i*4)+0] = layers[i].Opacity
			if layers[i].Frames().FrameAge > 10 {
				layerData[(i*4)+0] = 0.5
			}
			if layers[i].Frames().IsStill && !firstFrame {
				continue
			}
			layers[i].Frames().FrameAge += 1
			if !layers[i].Frames().IsReady {
				continue
			}
			rf := layers[i].Frames().LastFrame
			if rf == nil {
				continue
			}

			rendering.SendFrameToGPU(rf, layers[i].Frames().TextureIDs, int(i))
		}
		currentTheatre.Animate(float32(time.Since(deltaTimer).Nanoseconds()) * 1e-9)
		deltaTimer = time.Now()
		gl.Uniform4fv(layerDataUniform, numLayers, &layerData[0])
		gl.Uniform4fv(layerPosUniform, numLayers, &layerPos[0])

		gl.DrawArrays(gl.TRIANGLES, 0, 2*3)

		for _, stage := range nonWindowStages {
			// Switch to the framebuffer connected to the window
			frames := stage.Sink.Frames()
			gl.BindFramebuffer(gl.FRAMEBUFFER, frames.FramebufferID)
			gl.Viewport(0, 0, int32(frames.Width), int32(frames.Height))
			gl.Clear(gl.COLOR_BUFFER_BIT)
			layers = stage.Layers
			gl.Uniform1ui(stageDataUniform, stage.StageData())
			for i := range numLayers {
				layerPos[(i*4)+0] = layers[i].Position.X
				layerPos[(i*4)+1] = layers[i].Position.Y
				layerPos[(i*4)+2] = layers[i].Size.X
				layerPos[(i*4)+3] = layers[i].Size.Y
				layerData[(i*4)+0] = layers[i].Opacity
			}
			gl.Uniform4fv(layerDataUniform, numLayers, &layerData[0])
			gl.Uniform4fv(layerPosUniform, numLayers, &layerPos[0])

			gl.DrawArrays(gl.TRIANGLES, 0, 2*3)
			frame := frames.GetBlankFrame()
			gl.ReadPixels(0, 0, int32(frames.Width), int32(frames.Height), gl.RGB, gl.UNSIGNED_BYTE, gl.Ptr(frame.Data))
			frames.SendFrame(frame)
		}

		// Maintenance
		window.SwapBuffers()
		frameCounter++
		if time.Since(frameTimer) > 1*time.Second {
			if theApi != nil {
				theApi.FPS = frameCounter
			}
			frameCounter = 0
			frameTimer = time.Now()
		}
		glfw.PollEvents()
		firstFrame = false
	}
}
