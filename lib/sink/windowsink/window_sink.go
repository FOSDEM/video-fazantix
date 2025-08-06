package windowsink

import (
	"log"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type WindowSink struct {
	frames layer.FrameForwarder
	Window *glfw.Window
}

func New(name string, cfg *config.WindowSinkCfg, frameCfg *encdec.FrameCfg, alloc encdec.FrameAllocator) *WindowSink {
	w := &WindowSink{}
	w.frames.Init(
		name,
		&encdec.FrameInfo{
			FrameType: encdec.RGBFrames,
			PixFmt:    []uint8{},
			FrameCfg:  *frameCfg,
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
	w.Frames().Debug("Initializing window")
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

	vendor := gl.GoStr(gl.GetString(gl.VENDOR))
	renderer := gl.GoStr(gl.GetString(gl.RENDERER))
	version := gl.GoStr(gl.GetString(gl.VERSION))

	w.log("OpenGL version %s / %s / %s", vendor, renderer, version)

	return window
}

func (w *WindowSink) log(msg string, args ...interface{}) {
	w.Frames().Log(msg, args...)
}
