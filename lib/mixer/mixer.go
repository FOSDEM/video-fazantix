package mixer

import (
	"log"
	"runtime"
	"time"

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

func MakeWindowAndMix(cfg *config.Config) {
	alloc := &encdec.DumbFrameAllocator{}

	theatre, err := theatre.New(cfg, alloc)
	if err != nil {
		log.Fatalf("could not build theatre: %s", err)
	}

	initGL()
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

	layers := windowStage.Layers
	numLayers := int32(len(layers))

	glvars := rendering.AllocateGLVars(program, numLayers)

	// Create extra framebuffers as rendertargets
	nonWindowStages := theatre.NonWindowStageList

	for _, stage := range nonWindowStages {
		rendering.SetupTextures(stage.Sink.Frames())
		rendering.UseAsFramebuffer(stage.Sink.Frames())
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
		// Render
		gl.UseProgram(program)

		gl.BindVertexArray(glvars.VAO)

		layers = windowStage.Layers

		dt := time.Since(deltaTimer)
		deltaTimer = time.Now()

		// send frames
		for i := range numLayers {
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

		// push vars common for all stages
		glvars.PushCommonVars()

		glvars.DrawStage(windowStage)

		for _, stage := range nonWindowStages {
			glvars.DrawStage(stage)

			frames := stage.Sink.Frames()
			frame := frames.GetBlankFrame()
			gl.ReadPixels(0, 0, int32(frames.Width), int32(frames.Height), gl.RGB, gl.UNSIGNED_BYTE, gl.Ptr(frame.Data))
			frames.SendFrame(frame)
		}

		// Maintenance
		theatre.Animate(float32(dt.Nanoseconds()) * 1e-9)
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
