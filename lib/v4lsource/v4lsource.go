package v4lsource

import (
	"context"
	"log"
	"strings"

	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/fosdem/fazantix/lib/v4lsource/device"
	"github.com/fosdem/fazantix/lib/v4lsource/v4l2"
)

type V4LSource struct {
	path         string
	Format       string
	Device       *device.Device
	rawCamFrames <-chan []byte

	frames layer.FrameForwarder
	alloc  encdec.FrameAllocator

	requestedFrameCfg *encdec.FrameCfg
}

func New(name string, cfg *config.V4LSourceCfg, alloc encdec.FrameAllocator) *V4LSource {
	s := &V4LSource{}
	s.path = cfg.Path
	s.frames.Name = name
	s.alloc = alloc

	s.Format = cfg.Fmt

	s.requestedFrameCfg = &cfg.FrameCfg

	return s
}

func (s *V4LSource) Frames() *layer.FrameForwarder {
	return &s.frames
}

func (s *V4LSource) Start() bool {
	pixfmt := v4l2.PixelFmtYUYV
	switch strings.ToLower(s.Format) {
	case "mjpeg":
		pixfmt = v4l2.PixelFmtMJPEG
	case "yuyv":
		pixfmt = v4l2.PixelFmtYUYV
	}

	s.log("Loading v4l2 device %s", s.path)

	camera, err := device.Open(
		s.path,
		device.WithPixFormat(v4l2.PixFormat{
			PixelFormat: pixfmt,
			Width:       uint32(s.requestedFrameCfg.Width),
			Height:      uint32(s.requestedFrameCfg.Height),
		}),
	)
	if err != nil {
		s.log("Failed to open device: %s", err)
		return false
	}
	s.log("Opened device")

	fps, err := camera.GetFrameRate()
	if err != nil {
		s.log("Failed to get framerate: %s", err)
	}
	s.log("framerate: %d", fps)

	if err := camera.Start(context.TODO()); err != nil {
		s.log("camera start: %s", err)
	}
	s.rawCamFrames = camera.GetOutput()
	s.Device = camera
	// TODO: Wait until the device is actually streaming

	s.log("Got first frame")

	format, err := s.Device.GetPixFormat()
	if err != nil {
		s.log("Could not get pixfmt: %s", err)
	}

	s.log("format: %s", format)
	s.log(
		"requested resolution is %dx%d, actual is %dx%d",
		s.requestedFrameCfg.Width,
		s.requestedFrameCfg.Height,
		int(format.Width),
		int(format.Height),
	)

	frameCfg := encdec.FrameCfg{
		Width:              int(format.Width),
		Height:             int(format.Height),
		NumAllocatedFrames: s.requestedFrameCfg.NumAllocatedFrames,
	}

	switch strings.ToLower(s.Format) {
	case "mjpeg":
		dummyImg := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		s.frames.Init(
			s.frames.Name,
			&encdec.FrameInfo{
				FrameType: encdec.RGBAFrames,
				PixFmt:    dummyImg.Pix,
				FrameCfg:  frameCfg,
			},
			s.alloc,
		)
	case "yuyv":
		s.frames.Init(
			s.frames.Name,
			&encdec.FrameInfo{
				FrameType: encdec.RGBAFrames,
				PixFmt:    []uint8{},
				FrameCfg:  frameCfg,
			},
			s.alloc,
		)
	}

	go s.decodeFrames()
	return true
}

func (s *V4LSource) Stop() {
	err := s.Device.Close()
	if err != nil {
		log.Printf("Could not close device: %s", err)
		return
	}
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
		frame := s.frames.GetFrameForWriting()
		if frame == nil {
			continue // drop the frame as instructed
		}

		err := encdec.DecodeRGBfromImage(rawFrame, frame)
		if err != nil {
			s.log("Could not decode frame: %s", err)
			s.frames.FailedWriting(frame)
			continue
		}
		s.frames.FinishedWriting(frame)
	}
}

func (s *V4LSource) decodeFrames422p() {
	for rawFrame := range s.rawCamFrames {
		frame := s.frames.GetFrameForWriting()
		if frame == nil {
			continue // drop the frame as instructed
		}

		err := encdec.DecodeYUYV422(rawFrame, frame)
		if err != nil {
			s.log("Could not decode frame: %s", err)
			s.frames.FailedWriting(frame)
			continue
		}
		s.frames.FinishedWriting(frame)
	}
}

func (s *V4LSource) PixFmt() []uint8 {
	panic("why do you want this")
}

func (s *V4LSource) log(msg string, args ...interface{}) {
	s.Frames().Log(msg, args...)
}
