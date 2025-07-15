package rendering

import (
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type PBOFrameAllocator struct{}

func (p *PBOFrameAllocator) NewFrame(info *encdec.FrameInfo) *encdec.Frame {
	bufSize := info.CalcBufSize()

	frame := &encdec.Frame{
		Width:             info.Width,
		Height:            info.Height,
		Type:              info.FrameType,
		GLPixelBufferSize: uint32(bufSize),
	}

	var streamType uint32
	switch info.TransportType {
	case encdec.TransportToGPU:
		streamType = gl.STREAM_DRAW
	case encdec.TransportFromGPU:
		streamType = gl.STREAM_READ
	default:
		panic("unknown transport type")
	}

	switch info.FrameType {
	case encdec.RGBFrames:
		frame.GLPixelBufferType = gl.PIXEL_PACK_BUFFER
	case encdec.RGBAFrames:
		frame.GLPixelBufferType = gl.PIXEL_PACK_BUFFER
	case encdec.YUV422Frames:
		frame.GLPixelBufferType = gl.PIXEL_UNPACK_BUFFER
	default:
		panic("unknown frame type")
	}

	gl.GenBuffers(1, &frame.GLPixelBufferID)
	gl.BindBuffer(frame.GLPixelBufferType, frame.GLPixelBufferID)
	gl.BufferData(frame.GLPixelBufferType, bufSize, gl.Ptr(nil), streamType)
	if info.TransportType == encdec.TransportToGPU {
		mapBuffer(frame, gl.WRITE_ONLY)
	}
	gl.BindBuffer(frame.GLPixelBufferType, 0)

	return frame
}
