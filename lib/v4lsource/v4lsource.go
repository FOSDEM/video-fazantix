package v4lsource

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"syscall"
	"time"

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
	path   string
	Format string
	Device *device.Device

	frames layer.FrameForwarder

	requestedFrameCfg  *encdec.FrameCfg
	numFramesInWriting int
	framesInWriting    []*encdec.Frame

	hadValidFrame      bool
	brokenFrameCounter uint64
}

func New(name string, cfg *config.V4LSourceCfg) *V4LSource {
	s := &V4LSource{}
	s.path = cfg.Path
	s.Frames().Name = name

	s.Format = cfg.Fmt

	s.requestedFrameCfg = &cfg.FrameCfg
	s.numFramesInWriting = cfg.NumFramesInWriting

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

	s.log("Loading v4l2 device %s with %d frames in transit", s.path, s.requestedFrameCfg.NumAllocatedFrames)
	s.hadValidFrame = false

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
		device.WithBufferSize(uint32(s.requestedFrameCfg.NumAllocatedFrames)),
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

	if err := camera.InitForStreaming(); err != nil {
		s.log("camera start: %s", err)
	}
	s.Device = camera

	format, err := s.Device.GetPixFormat()
	if err != nil {
		s.log("Could not get pixfmt: %s", err)
	}

	s.log("format: %s", format)
	if s.requestedFrameCfg.Width != int(format.Width) || s.requestedFrameCfg.Height != int(format.Height) || (format.PixelFormat != pixfmt) {
		s.Frames().Error("driver changed format")
	}

	frameCfg := encdec.FrameCfg{
		Width:              int(format.Width),
		Height:             int(format.Height),
		NumAllocatedFrames: s.requestedFrameCfg.NumAllocatedFrames,
	}

	alloc := encdec.NewFixedFrameAllocator(s.Device.GetBuf)

	switch strings.ToLower(s.Format) {
	case "mjpeg":
		dummyImg := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		s.Frames().Init(
			s.Frames().Name,
			&encdec.FrameInfo{
				FrameType: encdec.RGBAFrames,
				PixFmt:    dummyImg.Pix,
				FrameCfg:  frameCfg,
			},
			alloc,
		)
	case "yuyv":
		s.Frames().Init(
			s.Frames().Name,
			&encdec.FrameInfo{
				FrameType: encdec.YUV422pFrames,
				PixFmt:    []uint8{},
				FrameCfg:  frameCfg,
			},
			alloc,
		)
	}

	go s.streamLoopLoop()

	return true
}

func (s *V4LSource) Stop() {
	err := s.Device.Close()
	if err != nil {
		log.Printf("Could not close device: %s", err)
		return
	}
}

func (s *V4LSource) finaliseFrame(frame *encdec.Frame) error {
	switch s.Format {
	case "mjpeg":
		return encdec.DecodeRGBfromImage(frame.Data, frame)
	case "yuyv":
		return encdec.PrepareYUYV(frame)
	}
	return fmt.Errorf("unknown format: %s", s.Format)
}

func (s *V4LSource) PixFmt() []uint8 {
	panic("why do you want this")
}

func (s *V4LSource) log(msg string, args ...interface{}) {
	s.Frames().Log(msg, args...)
}

func (s *V4LSource) enqueueFrames() error {
	s.framesInWriting = make([]*encdec.Frame, s.Device.BufferCount())

	for range s.numFramesInWriting {
		err := s.enqueueFrame()
		if err != nil {
			s.releaseFrames()
			return fmt.Errorf("error while enqueueing initial frames: %w", err)
		}
	}
	return nil
}

func (s *V4LSource) enqueueFrame() error {
	frame := s.Frames().GetFrameForWriting()
	if frame == nil {
		// Frame dropped, warning about this should be done by the frameforwarder
		return nil
	}
	s.framesInWriting[frame.SoulID] = frame

	_, err := v4l2.QueueBuffer(
		s.Device.Fd(),
		s.Device.MemIOType(),
		s.Device.BufferType(),
		uint32(frame.SoulID),
	)
	if err != nil {
		return fmt.Errorf("device: buffer queueing: %w", err)
	}
	return nil
}

func (s *V4LSource) releaseFrames() {
	for _, frame := range s.framesInWriting {
		if frame == nil {
			continue
		}
		s.Frames().FailedWriting(frame)
	}
}

func v4l2BufFlagsToStrings(flags uint32) []string {
	result := make([]string, 0)
	if flags&v4l2.BufFlagMapped > 0 {
		result = append(result, "mapped")
	}
	if flags&v4l2.BufFlagQueued > 0 {
		result = append(result, "queued")
	}
	if flags&v4l2.BufFlagDone > 0 {
		result = append(result, "done")
	}
	if flags&v4l2.BufFlagError > 0 {
		result = append(result, "error")
	}
	if flags&v4l2.BufFlagKeyFrame > 0 {
		result = append(result, "keyframe")
	}
	if flags&v4l2.BufFlagPFrame > 0 {
		result = append(result, "pframe")
	}
	if flags&v4l2.BufFlagBFrame > 0 {
		result = append(result, "bframe")
	}
	if flags&v4l2.BufFlagTimeCode > 0 {
		result = append(result, "timecode")
	}
	if flags&v4l2.BufFlagPrepared > 0 {
		result = append(result, "prepared")
	}
	if flags&v4l2.BufFlagNoCacheInvalidate > 0 {
		result = append(result, "no-cache-invalidate")
	}
	if flags&v4l2.BufFlagNoCacheClean > 0 {
		result = append(result, "no-cache-clean")
	}
	if flags&v4l2.BufFlagLast > 0 {
		result = append(result, "last")
	}
	return result
}

func (s *V4LSource) dequeueFrame() error {
	var buff v4l2.Buffer
	var err error
	for {
		buff, err = v4l2.DequeueBuffer(
			s.Device.Fd(),
			s.Device.MemIOType(),
			s.Device.BufferType(),
		)
		if err != nil {
			if errors.Is(err, syscall.EAGAIN) {
				continue
			}
			s.brokenFrameCounter++
			return fmt.Errorf("device: stream loop dequeue: %w", err)
		}
		break
	}

	frame := s.framesInWriting[buff.Index]

	if !s.v4l2BufOK(&buff) {
		s.brokenFrameCounter++
		s.Frames().FailedWriting(frame)
		return nil
	}

	if !s.hadValidFrame {
		s.hadValidFrame = true
		if s.brokenFrameCounter > 0 {
			s.Frames().Debug("Got %d invalid frames at start", s.brokenFrameCounter)
		}
	}
	if buff.Flags&v4l2.BufFlagMapped == 0 {
		// something really bad happened, restart the stream
		return fmt.Errorf("Got invalid buffer, flags %v", v4l2BufFlagsToStrings(buff.Flags))
	}

	frame.Data = frame.Data[:buff.BytesUsed]
	s.framesInWriting[buff.Index] = nil
	err = s.finaliseFrame(frame)
	if err != nil {
		s.Frames().FailedWriting(frame)
		return fmt.Errorf("could not prepare frame: %w", err)
	} else {
		s.Frames().FinishedWriting(frame)
	}
	return nil
}

func (s *V4LSource) streamLoopLoop() {
	for {
		err := s.streamLoop()
		s.Frames().Error("stream loop died, starting again in a second: %s", err)
		// Stop the streaming and let V4L clean up
		err = v4l2.StreamOff(s.Device)
		if err != nil {
			s.Frames().Error("could not turn off stream: %s; restarting will probably fail", err.Error())
		}
		time.Sleep(1 * time.Second)
	}
}

func (s *V4LSource) streamLoop() error {
	err := s.enqueueFrames()
	if err != nil {
		panic(fmt.Sprintf("could not enqueue frames: %s", err))
	}
	defer s.releaseFrames()

	if err := v4l2.StreamOn(s.Device); err != nil {
		return fmt.Errorf("device: stream on: %w", err)
	}

	for {
		err = s.dequeueFrame()
		if err != nil {
			return fmt.Errorf("could not dequeue frame: %w", err)
		}
		err = s.enqueueFrame()
		if err != nil {
			return fmt.Errorf("could not enqueue frame: %w", err)
		}
	}
}

func (s *V4LSource) v4l2BufOK(buff *v4l2.Buffer) bool {
	if buff.Flags&v4l2.BufFlagError != 0 {
		return false
	}
	if buff.BytesUsed == 0 {
		return false
	}
	if buff.BytesUsed != buff.Length {
		return false
	}
	if int(buff.Length) != (s.requestedFrameCfg.Width * s.requestedFrameCfg.Height * 2) {
		s.Frames().Error("Buffer size incorrect")
		return false
	}
	return true
}
