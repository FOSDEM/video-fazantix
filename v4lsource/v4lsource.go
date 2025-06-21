package v4lsource

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"

	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/vidmix/layer"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"

	"github.com/vladimirvivien/go4vl/device"
	"github.com/vladimirvivien/go4vl/v4l2"
)

type V4LSource struct {
	Name            string
	Format          string
	Device          *device.Device
	Fw              int
	Fh              int
	Frames          <-chan []byte
	outputFramesRGB chan *image.NRGBA
	outputFramesYUV chan *image.YCbCr

	LastImage    *image.NRGBA
	LastImageYUV *image.YCbCr

	isReady bool

	frameType layer.FrameType

	requestedWidth  int
	requestedHeight int

	texture [3]uint32
}

func New(devName string, mode string, width int, height int) *V4LSource {
	s := &V4LSource{}
	s.Name = devName

	log.Printf("[%s] Loading v4l2 device %s", s.Name)

	s.outputFramesRGB = make(chan *image.NRGBA)
	s.outputFramesYUV = make(chan *image.YCbCr)
	s.Format = mode

	s.texture[0] = s.setupYUVTexture(width, height)
	s.texture[1] = s.setupYUVTexture(width/2, height)
	s.texture[2] = s.setupYUVTexture(width/2, height)

	s.requestedWidth = width
	s.requestedHeight = height

	return s
}

func (s *V4LSource) Texture(idx int) uint32 {
	return s.texture[idx]
}

func (s *V4LSource) IsReady() bool {
	return s.isReady
}

func (s *V4LSource) IsStill() bool {
	return false
}

func (s *V4LSource) Width() int {
	return s.Fw
}

func (s *V4LSource) Height() int {
	return s.Fh
}

func (s *V4LSource) FrameType() layer.FrameType {
	return s.frameType
}

func (s *V4LSource) GenRGBFrames() <-chan *image.NRGBA {
	return s.outputFramesRGB
}

func (s *V4LSource) GenYUVFrames() <-chan *image.YCbCr {
	return s.outputFramesYUV
}

func (s *V4LSource) Start() bool {
	pixfmt := v4l2.PixelFmtYUYV
	switch strings.ToLower(s.Format) {
	case "mjpeg":
		pixfmt = v4l2.PixelFmtMJPEG
	case "yuyv":
		pixfmt = v4l2.PixelFmtYUYV
	}

	camera, err := device.Open(
		s.Name,
		device.WithPixFormat(v4l2.PixFormat{PixelFormat: pixfmt, Width: uint32(s.requestedWidth), Height: uint32(s.requestedHeight)}),
	)
	if err != nil {
		log.Printf("[%s] Failed to open device: %s", s.Name, err)
		return false
	}
	log.Printf("[%s] Opened device at %dx%d", s.Name, s.requestedWidth, s.requestedHeight)

	fps, err := camera.GetFrameRate()
	if err != nil {
		log.Printf("[%s] Failed to get framerate: %s", s.Name, err)
	}
	log.Printf("[%s] framerate: %d", s.Name, fps)

	if err := camera.Start(context.TODO()); err != nil {
		log.Fatalf("[%s] camera start: %s", s.Name, err)
	}
	s.Frames = camera.GetOutput()
	s.Device = camera
	go s.decodeFrames()
	return true
}

func (s *V4LSource) Stop() {
	s.Device.Close()
}

func (s *V4LSource) decodeFrames() {
	// TODO: Wait until the device is actually streaming

	log.Printf("[%s] Got first frame", s.Name)

	format, err := s.Device.GetPixFormat()
	if err != nil {
		log.Printf("[%s] Could not get pixfmt: %s", s.Name, err)
	}

	log.Printf("[%s] format: %s", s.Name, format)
	s.Fw = int(format.Width)
	s.Fh = int(format.Height)

	switch s.Format {
	case "mjpeg":
		s.decodeFramesJPEG()
	case "yuyv":
		s.decodeFrames422p()
	}
}

func (s *V4LSource) decodeFramesJPEG() {
	gl.GenTextures(1, &s.texture[0])
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, s.texture[0])
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
		s.isReady = true
		s.outputFramesRGB <- nrgba
	}
}

func (s *V4LSource) decodeFrames422p() {
	s.frameType = layer.YUVFrames

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
		s.isReady = true
		s.outputFramesYUV <- ycbr
	}
}

func (s *V4LSource) setupYUVTexture(width int, height int) uint32 {
	var id uint32
	gl.GenTextures(1, &id)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, id)
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
	return id
}
