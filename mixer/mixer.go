package mixer

import (
	"log"
	"maps"
	"runtime"
	"slices"
	"time"

	"github.com/fosdem/fazantix/api"
	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/kbdctl"
	"github.com/fosdem/fazantix/rendering"
	"github.com/fosdem/fazantix/rendering/shaders"
	"github.com/fosdem/fazantix/theatre"
	"github.com/fosdem/fazantix/windowsink"
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

func keyCallback(theatre *theatre.Theatre, stageName string) func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	scenes := slices.Sorted(maps.Keys(theatre.Scenes))
	names := make([]string, len(theatre.Scenes))
	for i, n := range scenes {
		names[i] = n
	}

	return func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
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
				if selected > len(theatre.Scenes)-1 {
					log.Printf("Scene %d out of range\n", selected)
					return
				}
				log.Printf("set scene %s\n", names[selected])
				err := theatre.SetScene(stageName, names[selected])
				if err != nil {
					log.Println(err)
					return
				}
			}
		}
	}
}

func initGL() {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Printf("OpenGL version '%s'", version)
}

func MakeWindowAndMix(cfg *config.Config) {
	alloc := &encdec.DumbFrameAllocator{}

	theatre, err := theatre.New(cfg, alloc)
	if err != nil {
		log.Fatalf("could not build theatre: %s", err)
	}

	// assume exactly one window stage for now
	windowStage := theatre.GetTheSingleWindowStage()
	windowSink := windowStage.Sink.(*windowsink.WindowSink)
	windowSink.Start()
	kbdctl.SetupShortcutKeys(theatre, windowSink)
	initGL()

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

	program, err := shaders.BuildGLProgram(theatre.ShaderData())
	if err != nil {
		log.Fatalf("could not init GL program: %s", err)
	}

	theatre.Start()

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

	layers := windowStage.Layers
	numLayers := int32(len(layers))

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

	frameCounter := 0
	frameTimer := time.Now()
	deltaTimer := time.Now()
	firstFrame := true
	for !windowSink.Window.ShouldClose() {
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
		theatre.Animate(float32(time.Since(deltaTimer).Nanoseconds()) * 1e-9)
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
		windowSink.Window.SwapBuffers()
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
