package mixer

import (
	"log"
	"time"

	"github.com/fosdem/fazantix/lib/api"
	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/kbdctl"
	"github.com/fosdem/fazantix/lib/rendering"
	"github.com/fosdem/fazantix/lib/rendering/shaders"
	"github.com/fosdem/fazantix/lib/theatre"
	"github.com/fosdem/fazantix/lib/utils"
	"github.com/fosdem/fazantix/lib/windowsink"
)

func MakeWindowAndMix(cfg *config.Config) {
	alloc := &encdec.DumbFrameAllocator{}

	theatre, err := theatre.New(cfg, alloc)
	if err != nil {
		log.Fatalf("could not build theatre: %s", err)
	}

	err = rendering.Init()
	if err != nil {
		log.Fatalf("could not initialise renderer: %s", err)
	}
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

	glvars := rendering.NewGLVars(program, int32(len(theatre.SourceList)))

	// Create extra framebuffers as rendertargets
	nonWindowStages := theatre.NonWindowStageList

	for _, stage := range nonWindowStages {
		rendering.SetupTextures(stage.Sink.Frames())
		rendering.UseAsFramebuffer(stage.Sink.Frames())
	}

	glvars.Start()

	frameCounter := 0
	frameTimer := time.Now()
	var deltaTimer utils.DeltaTimer
	for !theatre.ShutdownRequested {
		glvars.StartFrame()
		dt := deltaTimer.Next()

		rendering.SendFramesToGPU(theatre.SourceList, dt)

		glvars.DrawStage(windowStage)

		for _, stage := range nonWindowStages {
			glvars.DrawStage(stage)
			rendering.GetFrameFromGPUInto(stage.Sink)
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
		kbdctl.Poll()
	}
}
