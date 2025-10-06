//go:build omt

package omtsink

import (
	"time"

	"github.com/fosdem/fazantix/external/libomt"
	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
)

type OmtSink struct {
	name   string
	frames layer.FrameForwarder

	quality libomt.Quality
	send    *libomt.OmtSend
	frame   *libomt.OmtMediaFrame
}

func New(name string, cfg *config.OmtSinkCfg, frameCfg *encdec.FrameCfg, alloc encdec.FrameAllocator) *OmtSink {
	f := &OmtSink{name: cfg.Name, quality: libomt.QualityDefault}

	if cfg.Quality != "" {
		switch cfg.Quality {
		case "low":
			f.quality = libomt.QualityLow
		case "medium":
			f.quality = libomt.QualityMedium
		case "high":
			f.quality = libomt.QualityHigh
		default:
			f.quality = libomt.QualityDefault
		}
	}

	f.frames.Init(
		name,
		&encdec.FrameInfo{
			FrameType: encdec.BGRAFrames,
			FrameCfg:  *frameCfg,
		},
		alloc,
	)
	return f
}

func (f *OmtSink) Start() bool {
	send, err := libomt.OmtSendCreate(f.name, f.quality)
	if err != nil {
		panic(err)
	}
	f.send = send
	f.Frames().Log("Starting OMT sender: %s", f.send.GetAddress())

	f.frame = &libomt.OmtMediaFrame{
		Width:             f.frames.Width,
		Height:            f.frames.Height,
		Codec:             libomt.CodecBGRA,
		Timestamp:         -1,
		ColorSpace:        libomt.ColorSpaceBT709,
		Flags:             0,
		Stride:            f.frames.Width * 4,
		DataLength:        f.frames.Width * f.frames.Height * 4,
		FrameRateN:        60000,
		FrameRateD:        1000,
		AspectRatio:       float32(f.frames.Width) / float32(f.frames.Height), // Assume square pixels
		FrameMetadata:     nil,
		SampleRate:        0,
		Channels:          0,
		SamplesPerChannel: 0,
	}

	go f.sendFrames()

	return true
}

func (f *OmtSink) sendFrames() {
	interval := time.Now()
	for {
		frame := f.Frames().GetFrameForReading()
		if frame == nil {
			continue
		}
		f.send.Send(f.frame, frame.Data)
		f.Frames().FinishedReading(frame)
		if time.Since(interval).Seconds() > 1 {
			interval = time.Now()
			stats := f.send.GetVideoStatistics()
			f.Frames().Debug("%d connections, %.2f MB/s", f.send.Connections(), float64(stats.BytesSentSinceLast)/1024.0/1024.0)
		}
	}
}

func (f *OmtSink) Frames() *layer.FrameForwarder {
	return &f.frames
}

func (f *OmtSink) log(msg string, args ...interface{}) {
	f.Frames().Log(msg, args...)
}
