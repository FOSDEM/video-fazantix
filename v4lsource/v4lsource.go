package v4lsource

import (
	"context"
	"log"
	"strings"

	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/layer"

	"github.com/vladimirvivien/go4vl/device"
	"github.com/vladimirvivien/go4vl/v4l2"
)

type V4LSource struct {
	path         string
	Format       string
	Device       *device.Device
	rawCamFrames <-chan []byte

	frames layer.FrameForwarder

	name string

	requestedWidth  int
	requestedHeight int
}

func New(name string, cfg *config.V4LSourceCfg) *V4LSource {
	s := &V4LSource{}
	s.name = name
	s.path = cfg.Path

	s.Format = cfg.Fmt

	s.requestedWidth = cfg.W
	s.requestedHeight = cfg.H

	return s
}

func (s *V4LSource) Frames() *layer.FrameForwarder {
	return &s.frames
}

func (s *V4LSource) Name() string {
	return s.name
}

func (s *V4LSource) Start() bool {
	pixfmt := v4l2.PixelFmtYUYV
	switch strings.ToLower(s.Format) {
	case "mjpeg":
		pixfmt = v4l2.PixelFmtMJPEG
	case "yuyv":
		pixfmt = v4l2.PixelFmtYUYV
	}

	log.Printf("[%s] Loading v4l2 device %s", s.name, s.path)

	camera, err := device.Open(
		s.path,
		device.WithPixFormat(v4l2.PixFormat{PixelFormat: pixfmt, Width: uint32(s.requestedWidth), Height: uint32(s.requestedHeight)}),
	)
	if err != nil {
		log.Printf("[%s] Failed to open device: %s", s.name, err)
		return false
	}
	log.Printf("[%s] Opened device at %dx%d", s.name, s.requestedWidth, s.requestedHeight)

	fps, err := camera.GetFrameRate()
	if err != nil {
		log.Printf("[%s] Failed to get framerate: %s", s.name, err)
	}
	log.Printf("[%s] framerate: %d", s.name, fps)

	if err := camera.Start(context.TODO()); err != nil {
		log.Fatalf("[%s] camera start: %s", s.name, err)
	}
	s.rawCamFrames = camera.GetOutput()
	s.Device = camera
	// TODO: Wait until the device is actually streaming

	log.Printf("[%s] Got first frame", s.name)

	format, err := s.Device.GetPixFormat()
	if err != nil {
		log.Printf("[%s] Could not get pixfmt: %s", s.name, err)
	}

	log.Printf("[%s] format: %s", s.name, format)

	switch strings.ToLower(s.Format) {
	case "mjpeg":
		dummyImg := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		s.frames.Init(
			encdec.RGBFrames, dummyImg.Pix,
			int(format.Width), int(format.Height),
		)
	case "yuyv":
		s.frames.Init(encdec.YUV422Frames, []uint8{}, int(format.Width), int(format.Height))
	}

	go s.decodeFrames()
	return true
}

func (s *V4LSource) Stop() {
	s.Device.Close()
}

func (s *V4LSource) decodeFrames() {
	switch s.Format {
	case "mjpeg":
		s.decodeFramesJPEG()
	case "yuyv":
		s.decodeFrames422p()
	}
}

func (s *V4LSource) decodeFramesJPEG() {
	// this does not work, dunno why
	for rawFrame := range s.rawCamFrames {
		frame := s.frames.GetBlankFrame()
		err := encdec.DecodeRGBfromImage(rawFrame, frame)
		if err != nil {
			log.Printf("[%s] Could not decode frame: %s", s.name, err)
			continue
		}
		s.frames.IsReady = true
		s.frames.SendFrame(frame)
	}
}

func (s *V4LSource) decodeFrames422p() {
	for rawFrame := range s.rawCamFrames {
		frame := s.frames.GetBlankFrame()
		err := encdec.DecodeYUYV422(rawFrame, frame)
		if err != nil {
			log.Printf("[%s] Could not decode frame: %s", s.name, err)
			continue
		}
		s.frames.IsReady = true
		s.frames.SendFrame(frame)
	}
}

func (s *V4LSource) PixFmt() []uint8 {
	panic("why do you want this")
}
