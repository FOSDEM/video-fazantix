package mixer

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"

	"github.com/vladimirvivien/go4vl/device"
	"github.com/vladimirvivien/go4vl/v4l2"
)

type Coordinate struct {
	x float32
	y float32
}

type Mask struct {
	top    float32
	bottom float32
	left   float32
	right  float32
}

type Source struct {
	Name    string
	IsStill bool

	IsVisible bool
	IsReady   bool

	Size     Coordinate
	Position Coordinate
	Mask     Mask

	OutputWidth  int
	OutputHeight int

	Texture  [3]uint32
	IsPlanar bool

	Width   uint32
	Height  uint32
	Squeeze float32

	// V4L2 source
	Format    string
	Device    *device.Device
	Fw        int
	Fh        int
	Frames    <-chan []byte
	Images    chan *image.NRGBA
	ImagesYUV chan *image.YCbCr

	LastImage    *image.NRGBA
	LastImageYUV *image.YCbCr
}

func newSource(name string, width int, height int) Source {
	s := Source{Name: name, IsStill: true, IsVisible: false, IsReady: false}
	s.Size = Coordinate{x: 1.0, y: 1.0}
	s.Squeeze = 1.0
	s.IsPlanar = false
	s.OutputWidth = width
	s.OutputHeight = height
	s.Position = Coordinate{x: 0.5, y: 0.5}
	s.Mask = Mask{top: 0, bottom: 0, left: 0, right: 0}
	return s
}

func (s *Source) Move(x float32, y float32, size float32) {
	s.Position.x = x
	s.Position.y = y
	s.Size.x = size
	s.Size.y = size / s.Squeeze
}

func (s *Source) LoadStill(path string) bool {
	log.Printf("[%s] Loading %s", s.Name, path)
	imgFile, err := os.Open(path)
	if err != nil {
		log.Printf("[%s] Error: %s", s.Name, err)
		return false
	}

	img, _, err := image.Decode(imgFile)
	if err != nil {
		log.Printf("[%s] Error: %s", s.Name, err)
		return false
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		log.Printf("[%s] Unsupported stride", s.Name)
		return false
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)
	log.Printf("[%s] Size: %dx%d", s.Name, rgba.Bounds().Dx(), rgba.Bounds().Dy())

	s.setupRGBTexture(&s.Texture[0], rgba.Rect.Size().X, rgba.Rect.Size().Y, rgba.Pix)
	s.IsReady = true
	return true
}

func (s *Source) DecodeFrames() {
	// Wait until the device is actually streaming
	//frame := <- s.Frames
	//_ = frame

	log.Printf("[%s] Got first frame", s.Name)

	format, err := s.Device.GetPixFormat()
	if err != nil {
		log.Printf("[%s] Could not get pixfmt: %s", s.Name, err)
	}

	log.Printf("[%s] format: %s", s.Name, format)
	s.Fw = int(format.Width)
	s.Fh = int(format.Height)

	// Create appropriate textures

	switch s.Format {
	case "mjpeg":
		s.DecodeFramesJPEG()
	case "yuyv":
		s.DecodeFrames422p()
	}
}

func (s *Source) DecodeFramesJPEG() {
	gl.GenTextures(1, &s.Texture[0])
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, s.Texture[0])
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	borderColor := mgl32.Vec4{0, 0, 0, 0}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &borderColor[0])
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(s.Fw),
		int32(s.Fh),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		nil)

	var frame []byte
	for frame = range s.Frames {
		img, _, err := image.Decode(bytes.NewReader(frame))
		if err != nil {
			fmt.Printf("[%s] decode failure: %s", s.Name, err)
		}
		bounds := img.Bounds()
		nrgba := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		draw.Draw(nrgba, nrgba.Bounds(), img, bounds.Min, draw.Src)
		s.IsReady = true
		s.Images <- nrgba
	}
}

func (s *Source) DecodeFrames422p() {

	s.IsPlanar = true

	var frame []byte
	for frame = range s.Frames {
		ycbr := image.NewYCbCr(image.Rect(0, 0, s.Fw, s.Fh), image.YCbCrSubsampleRatio422)
		for i := range ycbr.Cb {
			ii := i * 4
			ycbr.Y[i*2] = frame[ii]
			ycbr.Y[i*2+1] = frame[ii+2]
			ycbr.Cb[i] = frame[ii+1]
			ycbr.Cr[i] = frame[ii+3]
		}
		s.IsReady = true
		s.ImagesYUV <- ycbr
	}
}

func (s *Source) setupYUVTexture(id *uint32, width int, height int) {
	gl.GenTextures(1, id)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, *id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	// this is to compenasate for floating-point errors on x==0/y==0
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	buf := make([]uint8, width*height)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RED,
		int32(width),
		int32(height),
		0,
		gl.RED,
		gl.UNSIGNED_BYTE,
		gl.Ptr(&buf[0]),
	)
}

func (s *Source) setupRGBTexture(id *uint32, width int, height int, texture []byte) {
	gl.GenTextures(1, id)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, *id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	borderColor := mgl32.Vec4{0, 0, 0, 0}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &borderColor[0])
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(width),
		int32(height),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(texture))
}

func (s *Source) LoadV4l2(devName string, mode string, width int, height int) bool {
	log.Printf("[%s] Loading v4l2 device %s", s.Name, devName)
	s.IsStill = false

	pixfmt := v4l2.PixelFmtYUYV
	switch strings.ToLower(mode) {
	case "mjpeg":
		pixfmt = v4l2.PixelFmtMJPEG
	case "yuyv":
		pixfmt = v4l2.PixelFmtYUYV
	}
	s.Format = mode
	camera, err := device.Open(
		devName,
		device.WithPixFormat(v4l2.PixFormat{PixelFormat: pixfmt, Width: uint32(width), Height: uint32(height)}),
	)
	if err != nil {
		log.Printf("[%s] Failed to open device: %s", s.Name, err)
		return false
	}
	// defer camera.Close()

	log.Printf("[%s] Opened device at %dx%d", s.Name, width, height)

	fps, err := camera.GetFrameRate()
	if err != nil {
		log.Printf("[%s] Failed to get framerate: %s", s.Name, err)
	}
	log.Printf("[%s] framerate: %d", s.Name, fps)

	s.setupYUVTexture(&s.Texture[0], width, height)
	s.setupYUVTexture(&s.Texture[1], width/2, height)
	s.setupYUVTexture(&s.Texture[2], width/2, height)

	s.Squeeze = (float32(width) / float32(height)) / (float32(s.Width) / float32(s.Height))
	if err := camera.Start(context.TODO()); err != nil {
		log.Fatalf("[%s] camera start: %s", s.Name, err)
	}
	s.Frames = camera.GetOutput()
	s.Images = make(chan *image.NRGBA)
	s.ImagesYUV = make(chan *image.YCbCr)
	s.Device = camera
	go s.DecodeFrames()
	return true
}
