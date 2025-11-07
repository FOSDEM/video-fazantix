//go:build omt

package omtsource

import (
	"github.com/fosdem/fazantix/external/libomt"
	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
)

type OmtSource struct {
	name       string
	frames     layer.FrameForwarder
	recv       *libomt.OmtReceive
	dummyFrame *encdec.Frame
}

func New(name string, cfg *config.OmtSourceCfg, alloc encdec.FrameAllocator) *OmtSource {
	f := &OmtSource{name: cfg.Name}
	frameInfo := &encdec.FrameInfo{
		FrameType: encdec.RGBAFrames,
		PixFmt:    []uint8{},
		FrameCfg:  cfg.FrameCfg,
	}

	f.frames.Init(name, frameInfo, alloc)
	f.dummyFrame = alloc.NewFrame(frameInfo)
	return f
}

func (f *OmtSource) Start() bool {
	recv, err := libomt.OmtReceiveCreate(f.name, libomt.Video, libomt.PreferredVideoFormatBGRA, libomt.ReceiveFlagsNone)
	if err != nil {
		panic("Could not create OMT receiver")
	}
	f.recv = recv
	go f.receiveLoop()
	return true
}

func (f *OmtSource) receiveLoop() {
	for {
		frame := f.frames.GetFrameForWriting()
		if frame == nil {
			// drop frame
			f.recv.Receive(libomt.Video, 33, f.dummyFrame.Data)
			continue
		}

		err := encdec.PrepareRGBA(frame)
		if err != nil {
			f.Frames().Error("Could not prepare YUV422 buffer: %s", err)
			f.frames.FailedWriting(frame)
			return
		}
		mf := f.recv.Receive(libomt.Video, 33, frame.Data)
		if mf == nil {
			f.frames.FailedWriting(frame)
		} else {
			f.frames.FinishedWriting(frame)
		}

	}
}

func (f *OmtSource) Frames() *layer.FrameForwarder {
	return &f.frames
}

func (f *OmtSource) log(msg string, args ...interface{}) {
	f.Frames().Log(msg, args...)
}
