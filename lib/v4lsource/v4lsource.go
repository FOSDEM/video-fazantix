package v4lsource

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/fazantix/external/go4vl/device"
	"github.com/fosdem/fazantix/external/go4vl/v4l2"
	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/fosdem/fazantix/lib/utils"
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

func New(name string, cfg *config.V4LSourceCfg) *V4LSource {
	s := &V4LSource{}
	s.path = cfg.Path
	s.frames.Name = name
	s.alloc = &NullFrameAllocator{}

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

	if !strings.HasPrefix(s.path, "/") {
		lookup, err := utils.LocateUSBDevice(s.path)
		if err != nil {
			s.log("Failed to find device: %s", err)
			return false
		}
		v4l2dev := lookup.GetFirst(utils.V4L2Device)
		if v4l2dev == nil {
			s.log("USB device in port %s does not have a V4L2 driver", s.path)
			return false
		}
		s.log("Found V4L2 device %s at port %s", v4l2dev.Path, s.path)
		s.path = v4l2dev.Path
	}
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
				FrameType: encdec.YUV422pFrames,
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
		_ = encdec.PrepareYUYV(frame)
		copy(frame.Data, rawFrame)
		s.frames.FinishedWriting(frame)
	}
}

func (s *V4LSource) PixFmt() []uint8 {
	panic("why do you want this")
}

func (s *V4LSource) log(msg string, args ...interface{}) {
	s.Frames().Log(msg, args...)
}

func (s *V4LSource) enqueueFrames() error {
	numFrames := s.Frames().AvailableFramesForWriting()
	s.framesInWriting = make([]*encdec.Frame, numFrames)
	// Initial enqueue of buffers for capture
	for i := range numFrames {
		frame := s.frames.GetFrameForWriting()
		s.framesInWriting[i] = frame

		_, err := v4l2.QueueBuffer(fd, ioType, bufType, uint32(i))
		if err != nil {
			s.releaseFrames()
			return fmt.Errorf("device: buffer queueing: %w", err)
		}
	}
	return nil
}

func (s *V4LSource) releaseFrames() {
	for _, frame := range s.framesInWriting {
		s.Frames().FailedWriting(frame)
	}
}

func (s *V4LSource) startStreamLoop(ctx context.Context) error {
	dev := s.Device
	ioType := dev.MemIOType()
	bufType := dev.BufferType()
	fd := dev.Fd()

	if err := v4l2.StreamOn(d); err != nil {
		return fmt.Errorf("device: stream on: %w", err)
	}

	go func() {
		defer close(dev.output)

		err := s.enqueueFrames()
		if err != nil {
			panic(fmt.Sprintf("could not enqueue frames: %s", err))
		}
		defer s.releaseFrames()

		fd := dev.Fd()
		var frame []byte
		waitForRead := v4l2.WaitForRead(d)
		for {
			select {
			// handle stream capture (read from driver)
			case <-waitForRead:
				buff, err := v4l2.DequeueBuffer(fd, ioMemType, bufType)
				if err != nil {
					if errors.Is(err, sys.EAGAIN) {
						continue
					}
					panic(fmt.Sprintf("device: stream loop dequeue: %s", err))
				}

				// copy mapped buffer (copying avoids polluted data from subsequent dequeue ops)
				if buff.Flags&v4l2.BufFlagMapped != 0 && buff.Flags&v4l2.BufFlagError == 0 {
					frame = make([]byte, buff.BytesUsed)
					if n := copy(frame, dev.buffers[buff.Index][:buff.BytesUsed]); n == 0 {
						dev.output <- []byte{}
					}
					dev.output <- frame
					frame = nil
				} else {
					dev.output <- []byte{}
				}

				if _, err := v4l2.QueueBuffer(fd, ioMemType, bufType, buff.Index); err != nil {
					panic(fmt.Sprintf("device: stream loop queue: %s: buff: %#v", err, buff))
				}
			case <-ctx.Done():
				dev.Stop()
				return
			}
		}
	}()

	return nil
}
