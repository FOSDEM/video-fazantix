package main

import (
	"flag"
	"log"
	"runtime"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/fosdem/fazantix/lib/rendering"
	"github.com/fosdem/fazantix/lib/rendering/shaders"
	"github.com/fosdem/fazantix/lib/sink/windowsink"
	"github.com/fosdem/fazantix/lib/source/stdinsource"
	"github.com/fosdem/fazantix/lib/utils"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func main() {

	titlePtr := flag.String("title", "WindowSink", "Window title for this sink")
	widthPtr := flag.Uint("width", 1920, "Width of the window sink")
	heightPtr := flag.Uint("height", 1080, "Height of the window sink")
	ratePtr := flag.Uint("rate", 60, "Framerate limiter on the input data")
	flag.Parse()

	width := int(*widthPtr)
	height := int(*heightPtr)
	rate := int(*ratePtr)
	runtime.LockOSThread()
	log.Printf("Accepting %dx%d@%d\n", width, height, rate)

	alloc := &encdec.DumbFrameAllocator{}
	err := rendering.Init()
	if err != nil {
		log.Fatalf("could not initialise renderer: %s", err)
	}
	sinkCfg := &config.WindowSinkCfg{}
	frameCfg := &encdec.FrameCfg{
		Width:              width,
		Height:             height,
		NumAllocatedFrames: 2,
	}
	windowSink := windowsink.New(*titlePtr, sinkCfg, frameCfg, alloc)
	windowSink.Start()
	stdinSource := stdinsource.New(width, height, rate, alloc)
	sources := make([]layer.Source, 1)
	sources[0] = stdinSource
	stdinSource.Start()
	shaderData := &shaders.ShaderData{
		Sources:        sources,
		NumSources:     1,
		NumLayers:      1,
		FallbackColour: utils.ColourParse("#ff0000"),
	}
	program, err := shaders.BuildGLProgram(shaderData)
	if err != nil {
		log.Fatalf("could not init GL program: %s", err)
	}
	glvars := rendering.NewGLVars(
		program, 1,
		sources, []int32{-1},
		utils.ColourParse("#ff0000"),
	)

	stage := &layer.Stage{
		Layers:        make([]*layer.Layer, 1),
		DefaultScene:  "stdin",
		Sink:          windowSink,
		SourceIndices: []int32{0},
		SourceTypes:   []encdec.FrameType{encdec.RGBAFrames},
	}
	stage.Layers[0] = layer.New(0, stdinSource, width, height)
	stage.Layers[0].Position.X = 0.0
	stage.Layers[0].Position.Y = 0.0
	stage.Layers[0].Opacity = 1
	glvars.Start()
	rendering.SetupTextures(stdinSource.Frames())

	var deltaTimer utils.DeltaTimer
	for {
		glvars.StartFrame()
		dt := deltaTimer.Next()
		rendering.SendFramesToGPU(sources, dt)
		glvars.DrawStage(stage)
		windowSink.Window.SwapBuffers()
		glfw.PollEvents()
	}
}
