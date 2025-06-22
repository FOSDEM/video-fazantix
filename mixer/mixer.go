package mixer

import (
	"log"
	"runtime"

	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/ffmpegsource"
	"github.com/fosdem/fazantix/imgsource"
	"github.com/fosdem/fazantix/layer"
	"github.com/fosdem/fazantix/rendering/shaders"
	"github.com/fosdem/fazantix/v4lsource"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

const windowWidth = 1920
const windowHeight = 1080
const numLayers = 3
const f32 = 4

var (
	layers [3]*layer.Layer
)

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

func makeWindow() *glfw.Window {
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
	window, err := glfw.CreateWindow(windowWidth, windowHeight, "OpenGL", nil, nil)
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

func MakeWindowAndMix() {
	window := makeWindow()
	initGL()

	shaderer, err := shaders.BuildShaderer()
	if err != nil {
		log.Fatalf("Could not init shaders: %s", err)
	}

	allLayers := map[string]*layer.Layer{
		"balloon": layer.New(
			"background",
			imgsource.New("background.png"),
			windowWidth, windowHeight,
		),
		"balloon2": layer.New(
			"background",
			imgsource.New("background.png"),
			windowWidth, windowHeight,
		),
		"balloon3": layer.New(
			"background",
			imgsource.New("background.png"),
			windowWidth, windowHeight,
		),
		"pheasants": layer.New(
			"sauce",
			ffmpegsource.New(
				`
			ffmpeg -stream_loop -1 -re -i ~/s/random_shit/test_videos/pheasants.webm -vf scale=1920:1080 -pix_fmt yuv422p -f rawvideo -r 60 -
			`,
				1920, 1080,
			),
			windowWidth, windowHeight,
		),
		"fazant": layer.New(
			"sauce",
			ffmpegsource.New(
				`
			ffmpeg -stream_loop -1 -re -i ~/s/random_shit/test_videos/fazantfazantfazant.mkv -vf scale=1920:1080 -pix_fmt yuv422p -f rawvideo -r 60 -
			`,
				1920, 1080,
			),
			windowWidth, windowHeight,
		),
		"video0": layer.New(
			"slides",
			v4lsource.New("/dev/video0", "yuyv", 1920, 1080),
			windowWidth, windowHeight,
		),
		"video0_ffmpeg": layer.New(
			"slides",
			ffmpegsource.New(
				`
ffmpeg -f v4l2 -framerate 60 -video_size 1920x1080 -i /dev/video0 -pix_fmt yuv422p -f rawvideo -r 60 -
			`,
				1920, 1080,
			),
			windowWidth, windowHeight,
		),
		"video4": layer.New(
			"slides",
			v4lsource.New("/dev/video4", "yuyv", 1920, 1080),
			windowWidth, windowHeight,
		),
		"video4_ffmpeg": layer.New(
			"slides",
			ffmpegsource.New(
				`
ffmpeg -f v4l2 -framerate 60 -video_size 1920x1080 -i /dev/video4 -pix_fmt yuv422p -f rawvideo -r 60 -
			`,
				1920, 1080,
			),
			windowWidth, windowHeight,
		),
		"cows": layer.New(
			"sauce",
			ffmpegsource.New(
				`
			ffmpeg -stream_loop -1 -re -i ~/s/random_shit/test_videos/cows.mp4 -vf scale=1920:1080 -pix_fmt yuv422p -f rawvideo -r 60 -
			`,
				1920, 1080,
			),
			windowWidth, windowHeight,
		),
	}

	layers[0] = allLayers["balloon"]
	layers[1] = allLayers["balloon2"]
	layers[2] = allLayers["balloon3"]

	layers[0].Source.Start()
	layers[0].Move(0, 0, 1)

	layers[1].Source.Start()
	layers[1].Move(0.025, 0.049, 0.79)

	layers[2].Source.Start()
	layers[2].Move(0.75, 0.6, 0.2)

	for i := range layers {
		layers[i].SetupTextures()
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

	vertAttrib := uint32(gl.GetAttribLocation(shaderer.Program, gl.Str("position\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointerWithOffset(vertAttrib, 2, gl.FLOAT, false, stride, 0)

	texCoordAttrib := uint32(gl.GetAttribLocation(shaderer.Program, gl.Str("uv\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointerWithOffset(texCoordAttrib, 2, gl.FLOAT, false, stride, 2*f32)

	var layerPos [numLayers * 4]float32
	layerPosUniform := gl.GetUniformLocation(shaderer.Program, gl.Str("sourcePosition\x00"))
	gl.Uniform4fv(layerPosUniform, numLayers, &layerPos[0])

	// Allocate 3 textures for every layer in case of planar YUV
	const numTextures = numLayers * 3
	var textures [numTextures]int32
	for i := range numTextures {
		textures[i] = int32(i)
	}
	texUniform := gl.GetUniformLocation(shaderer.Program, gl.Str("tex\x00"))
	gl.Uniform1iv(texUniform, numTextures, &textures[0])

	gl.ClearColor(1.0, 0.0, 0.0, 1.0)

	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Render
		gl.UseProgram(shaderer.Program)

		gl.BindVertexArray(vao)

		gl.Uniform1iv(texUniform, numTextures, &textures[0])
		for i := range numLayers {
			if !layers[i].Frames().IsReady {
				continue
			}
			rf := layers[i].Frames().LastFrame
			if rf == nil {
				continue
			}

			channelType := uint32(gl.RED)
			if rf.Type == encdec.RGBFrames {
				channelType = gl.RGBA
			}

			for j := 0; j < rf.NumTextures; j++ {
				dataPtr, w, h := rf.Texture(j)

				gl.ActiveTexture(uint32(gl.TEXTURE0 + (i * 3) + j))
				gl.BindTexture(gl.TEXTURE_2D, layers[i].TextureIDs[j])
				gl.TexSubImage2D(
					gl.TEXTURE_2D,
					0, 0, 0,
					int32(w), int32(h),
					channelType, gl.UNSIGNED_BYTE, gl.Ptr(dataPtr),
				)
			}

			layerPos[(i*4)+0] = layers[i].Position.X
			layerPos[(i*4)+1] = layers[i].Position.Y
			layerPos[(i*4)+2] = layers[i].Size.X
			layerPos[(i*4)+3] = layers[i].Size.Y

			layers[i].Frames().RecycleFrame(rf)
		}
		gl.Uniform4fv(layerPosUniform, numLayers, &layerPos[0])

		gl.DrawArrays(gl.TRIANGLES, 0, 2*3)

		// Maintenance
		window.SwapBuffers()
		glfw.PollEvents()
	}
}
