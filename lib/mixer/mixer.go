package mixer

import (
	"log"

	"github.com/fosdem/fazantix/lib/api"
	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/kbdctl"
	"github.com/fosdem/fazantix/lib/rendering"
	"github.com/fosdem/fazantix/lib/rendering/shaders"
	"github.com/fosdem/fazantix/lib/theatre"
	"github.com/fosdem/fazantix/lib/utils"
	"github.com/go-gl/gl/v4.1-core/gl"
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

	theatre.Start()

	api := api.ServeInBackground(theatre, cfg.Api)

	err = theatre.ResetToDefaultScenes()
	if err != nil {
		log.Fatalf("could not init default scenes: %s", err)
	}

	program, err := shaders.BuildGLProgram(theatre.ShaderData())
	if err != nil {
		log.Fatalf("could not init GL program: %s", err)
	}

	glvars := rendering.NewGLVars(program, int32(len(theatre.SourceList)))

	if len(theatre.WindowSinkList) > 1 {
		log.Fatalf("multiple window sinks are not supported yet")
		// TODO: figure out how to share the stuff managed by glvars between windows
	}
	if len(theatre.WindowSinkList) < 1 {
		log.Fatalf("usage without a window sink is not supported yet")
		// TODO: figure out how to make a GL context without a window
	}

	for _, sink := range theatre.WindowSinkList {
		kbdctl.SetupShortcutKeys(theatre, sink)
	}

	for _, stage := range theatre.NonWindowStageList {
		rendering.SetupTextures(stage.Sink.Frames())
		rendering.UseAsFramebuffer(stage.Sink.Frames())
	}

	glvars.Start()

	var deltaTimer utils.DeltaTimer
	for !theatre.ShutdownRequested {
		glvars.StartFrame()
		dt := deltaTimer.Next()

		rendering.SendFramesToGPU(theatre.SourceList, dt)

		for _, stage := range theatre.WindowStageList {
			glvars.DrawStage(stage)
		}

		for _, stage := range theatre.NonWindowStageList {
			readPBO, writePBO := stage.Sink.Frames().GetPixelBufferIndexes()
			gl.BindBuffer(gl.PIXEL_PACK_BUFFER, writePBO)
			glvars.DrawStage(stage) // gl.BindFramebuffer
			gl.ReadPixels(0, 0, int32(stage.Sink.Frames().Width), int32(stage.Sink.Frames().Height), gl.RGB, gl.UNSIGNED_BYTE, gl.PtrOffset(0))
			gl.BindBuffer(gl.PIXEL_PACK_BUFFER, readPBO)
			rendering.GetFrameFromGPUInto(stage.Sink)
			gl.BindBuffer(gl.PIXEL_PACK_BUFFER, 0)
		}

		for _, sink := range theatre.WindowSinkList {
			sink.Window.SwapBuffers()
			if sink.Window.ShouldClose() {
				theatre.ShutdownRequested = true
			}
		}

		// Maintenance
		theatre.Animate(float32(dt.Nanoseconds()) * 1e-9)
		api.Stats.Update()
		kbdctl.Poll()
	}
}
