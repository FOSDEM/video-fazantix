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

	"github.com/vladimirvivien/go4vl/device"
	"github.com/vladimirvivien/go4vl/v4l2"
)

type V4LSource struct {
	Name         string
	Format       string
	Device       *device.Device
	rawCamFrames <-chan []byte

	isReady bool

	frames layer.FrameForwarder

	requestedWidth  int
	requestedHeight int
	frameWidth      int
	frameHeight     int
}

func New(devName string, mode string, width int, height int) *V4LSource {
	s := &V4LSource{}
	s.Name = devName

	log.Printf("[%s] Loading v4l2 device %s", s.Name)

	s.Format = mode

	s.requestedWidth = width
	s.requestedHeight = height
	s.frames.Init()

	return s
}

func (s *V4LSource) IsReady() bool {
	return s.isReady
}

func (s *V4LSource) IsStill() bool {
	return false
}

func (s *V4LSource) Width() int {
	return s.frameWidth
}

func (s *V4LSource) Height() int {
	return s.frameHeight
}

func (s *V4LSource) Frames() *layer.FrameForwarder {
	return &s.frames
}

func (s *V4LSource) Start() bool {
	pixfmt := v4l2.PixelFmtYUYV
	switch strings.ToLower(s.Format) {
	case "mjpeg":
		s.frames.FrameType = layer.RGBFrames
		pixfmt = v4l2.PixelFmtMJPEG
		dummyImg := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		s.frames.PixFmt = dummyImg.Pix
	case "yuyv":
		s.frames.FrameType = layer.YUV422Frames
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
	s.rawCamFrames = camera.GetOutput()
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
	s.frameWidth = int(format.Width)
	s.frameHeight = int(format.Height)

	switch s.Format {
	case "mjpeg":
		s.decodeFramesJPEG()
	case "yuyv":
		s.decodeFrames422p()
	}
}

func (s *V4LSource) decodeFramesJPEG() {
	// this does not work, dunno why
	var frame []byte
	for frame = range s.rawCamFrames {
		img, _, err := image.Decode(bytes.NewReader(frame))
		if err != nil {
			fmt.Printf("[%s] decode failure: %s", s.Name, err)
			continue
		}
		bounds := img.Bounds()
		nrgba := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		draw.Draw(nrgba, nrgba.Bounds(), img, bounds.Min, draw.Src)
		s.isReady = true
		s.frames.SendRGBFrame(nrgba)
	}
}

func (s *V4LSource) decodeFrames422p() {
	for frame := range s.rawCamFrames {
		ycbr := image.NewYCbCr(image.Rect(0, 0, s.frameWidth, s.frameHeight), image.YCbCrSubsampleRatio422)
		if len(frame) < len(ycbr.Cb)*4 {
			fmt.Printf("[%s] got a frame of len %d when %d was expected", s.Name, len(frame), len(ycbr.Cb)*4)
			continue
		}
		for i := range ycbr.Cb {
			j := i * 4
			ycbr.Y[i*2] = frame[j]
			ycbr.Y[i*2+1] = frame[j+2]
			ycbr.Cb[i] = frame[j+1]
			ycbr.Cr[i] = frame[j+3]
		}
		s.isReady = true
		s.frames.SendYUV422Frame(ycbr)
	}
}

func (s *V4LSource) PixFmt() []uint8 {
	panic("why do you want this")
}
