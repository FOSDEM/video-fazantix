package mixer

import (
	"log"
	"runtime"
	"time"
	"unsafe"

	"github.com/fosdem/fazantix/lib/api"
	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/kbdctl"
	"github.com/fosdem/fazantix/lib/rendering"
	"github.com/fosdem/fazantix/lib/rendering/shaders"
	"github.com/fosdem/fazantix/lib/theatre"
	"github.com/fosdem/fazantix/lib/windowsink"
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

func initGL() {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Printf("OpenGL version '%s'", version)
}

func HandleOpenGLError() {
	for {
		glerr := gl.GetError()
		if glerr != 0 {
			log.Printf("OpenGL error %d\n", glerr)
		} else {
			return
		}
	}
}

func OpenGLErrorPrinter(source uint32, gltype uint32, id uint32, severity uint32, length int32, message string, userParam unsafe.Pointer) {
	log.Printf("OpenGL error: %d %s\n", source, message)
}

func MakeWindowAndMix(cfg *config.Config) {
	alloc := &encdec.DumbFrameAllocator{}

	theatre, err := theatre.New(cfg, alloc)
	if err != nil {
		log.Fatalf("could not build theatre: %s", err)
	}

	initGL()

	gl.Enable(gl.DEBUG_OUTPUT)
	gl.Enable(gl.DEBUG_OUTPUT_SYNCHRONOUS)
	gl.DebugMessageCallback(OpenGLErrorPrinter, gl.Ptr(nil))

	// assume exactly one window stage for now
	windowStage := theatre.GetTheSingleWindowStage()
	windowSink := windowStage.Sink.(*windowsink.WindowSink)

	theatre.Start()
	kbdctl.SetupShortcutKeys(theatre, windowSink)

	api := api.ServeInBackground(theatre, cfg.Api)

	program, err := shaders.BuildGLProgram(theatre.ShaderData())
	if err != nil {
		log.Fatalf("could not init GL program: %s", err)
	}
	gl.UseProgram(program)

	// Configure the vertex data
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*f32, gl.Ptr(vertices), gl.STATIC_DRAW)
	HandleOpenGLError()

	stride := int32(4 * f32)

	vertAttrib := uint32(gl.GetAttribLocation(program, gl.Str("position\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointerWithOffset(vertAttrib, 2, gl.FLOAT, false, stride, 0)
	HandleOpenGLError()

	texCoordAttrib := uint32(gl.GetAttribLocation(program, gl.Str("uv\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointerWithOffset(texCoordAttrib, 2, gl.FLOAT, false, stride, 2*f32)
	HandleOpenGLError()

	layers := windowStage.Layers
	numLayers := int32(len(layers))

	layerPos := make([]float32, numLayers*4)
	layerPosUniform := gl.GetUniformLocation(program, gl.Str("layerPosition\x00"))
	HandleOpenGLError()
	gl.Uniform4fv(layerPosUniform, numLayers, &layerPos[0])
	HandleOpenGLError()

	layerData := make([]float32, numLayers*4)
	layerDataUniform := gl.GetUniformLocation(program, gl.Str("layerData\x00"))
	gl.Uniform4fv(layerDataUniform, numLayers, &layerData[0])
	HandleOpenGLError()

	stageDataUniform := gl.GetUniformLocation(program, gl.Str("stageData\x00"))
	gl.Uniform1ui(stageDataUniform, 0)
	HandleOpenGLError()

	// Allocate 3 textures for every layer in case of planar YUV
	numTextures := numLayers * 3
	textures := make([]int32, numTextures)
	for i := range numTextures {
		textures[i] = int32(i)
	}
	texUniform := gl.GetUniformLocation(program, gl.Str("tex\x00"))
	gl.Uniform1iv(texUniform, numTextures, &textures[0])

	// Create extra framebuffers as rendertargets
	nonWindowStages := theatre.NonWindowStageList

	for _, stage := range nonWindowStages {
		stage.Sink.Frames().SetupTextures()
		stage.Sink.Frames().UseAsFramebuffer()
	}

	for name, stage := range theatre.Stages {
		err := theatre.SetScene(name, stage.DefaultScene)
		if err != nil {
			log.Fatalf("Could not apply default scene: %s", err)
		}
	}

	gl.ClearColor(1.0, 0.0, 0.0, 1.0)
	HandleOpenGLError()

	frameCounter := 0
	frameTimer := time.Now()
	deltaTimer := time.Now()
	firstFrame := true
	for !windowSink.Window.ShouldClose() {
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		gl.Viewport(0, 0, int32(windowStage.Sink.Frames().Width), int32(windowStage.Sink.Frames().Height))
		gl.Clear(gl.COLOR_BUFFER_BIT)
		HandleOpenGLError()

		// Render
		gl.BindVertexArray(vao)
		HandleOpenGLError()

		gl.Uniform1iv(texUniform, numTextures, &textures[0])
		gl.Uniform1ui(stageDataUniform, windowStage.StageData())
		layers = windowStage.Layers

		dt := time.Since(deltaTimer)
		deltaTimer = time.Now()
		HandleOpenGLError()

		for i := range numLayers {
			layerPos[(i*4)+0] = layers[i].Position.X
			layerPos[(i*4)+1] = layers[i].Position.Y
			layerPos[(i*4)+2] = layers[i].Size.X
			layerPos[(i*4)+3] = layers[i].Size.Y
			layerData[(i*4)+0] = layers[i].Opacity
			layers[i].Frames().Age(dt)
			if layers[i].Frames().IsStill && !firstFrame {
				continue
			}

			frame := layers[i].Frames().GetFrameForReading()
			if frame == nil {
				continue
			}
			rendering.SendFrameToGPU(frame, layers[i].Frames().TextureIDs, int(i))
			layers[i].Frames().FinishedReading(frame)
		}
		theatre.Animate(float32(dt.Nanoseconds()) * 1e-9)
		gl.Uniform4fv(layerDataUniform, numLayers, &layerData[0])
		gl.Uniform4fv(layerPosUniform, numLayers, &layerPos[0])

		gl.DrawArrays(gl.TRIANGLES, 0, 2*3)
		HandleOpenGLError()

		for _, stage := range nonWindowStages {
			// Switch to the framebuffer connected to the window
			frames := stage.Sink.Frames()
			gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, frames.FramebufferID)
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
			HandleOpenGLError()
			frame := frames.GetBlankFrame()
			gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, 0)
			HandleOpenGLError()
			gl.BindFramebuffer(gl.READ_FRAMEBUFFER, frames.FramebufferID)
			HandleOpenGLError()
			frame.FramebufferToPBO()
			gl.BindFramebuffer(gl.READ_FRAMEBUFFER, 0)
			frames.SendFrame(frame)
		}

		// Maintenance
		windowSink.Window.SwapBuffers()
		frameCounter++
		if time.Since(frameTimer) > 1*time.Second {
			if api != nil {
				api.FPS = frameCounter
			}
			frameCounter = 0
			frameTimer = time.Now()
		}
		glfw.PollEvents()
		firstFrame = false
	}
}
