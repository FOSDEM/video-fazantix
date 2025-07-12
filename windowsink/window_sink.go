package windowsink

import (
	"log"

	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/layer"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type WindowSink struct {
	frames layer.FrameForwarder
	Window *glfw.Window
}

func New(name string, cfg *config.WindowSinkCfg, alloc encdec.FrameAllocator) *WindowSink {
	w := &WindowSink{}
	w.frames.Init(
		name,
		&encdec.FrameInfo{
			FrameType: encdec.RGBFrames,
			PixFmt:    []uint8{},
			FrameCfg:  cfg.FrameCfg,
		},
		alloc,
	)
	return w
}

func (w *WindowSink) Start() bool {
	if w.Window == nil {
		w.Window = w.makeWindow()
	}
	return true
}

func (w *WindowSink) Frames() *layer.FrameForwarder {
	return &w.frames
}

func (w *WindowSink) makeWindow() *glfw.Window {
	log.Println("Initializing window")
	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	window, err := glfw.CreateWindow(w.Frames().Width, w.Frames().Height, w.Frames().Name, nil, nil)
	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()

	return window
}
