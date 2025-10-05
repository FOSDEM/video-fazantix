package libomt

/*
#cgo pkg-config: libomt
#include "libomt.h"
#include "string.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type OmtSend struct {
	send *C.omt_send_t
	mf   *C.OMTMediaFrame
}

func OmtSendCreate(name string, quality Quality) (*OmtSend, error) {
	result := &OmtSend{}
	nameStr := C.CString(name)
	result.send = C.omt_send_create(nameStr, C.OMTQuality(quality))
	if result.send == nil {
		return nil, fmt.Errorf("failed to create OmtSend")
	}
	return result, nil
}

func (s *OmtSend) Send(frame *OmtMediaFrame, data []byte) int {
	if s.mf == nil {
		s.mf = &C.OMTMediaFrame{}
		s.mf.Data = C.malloc(C.size_t(len(data)))
		s.mf.Timestamp = C.int64_t(frame.Timestamp)
	}
	s.mf.Type = C.OMTFrameType_Video
	s.mf.Codec = uint32(frame.Codec)
	s.mf.Width = C.int(frame.Width)
	s.mf.Height = C.int(frame.Height)
	s.mf.Stride = C.int(frame.Stride)
	s.mf.Flags = uint32(frame.Flags)
	s.mf.FrameRateN = C.int(frame.FrameRateN)
	s.mf.FrameRateD = C.int(frame.FrameRateD)
	s.mf.AspectRatio = C.float(frame.AspectRatio)
	s.mf.ColorSpace = uint32(frame.ColorSpace)

	s.mf.DataLength = C.int(len(data))
	C.memcpy(s.mf.Data, unsafe.Pointer(&data[0]), C.size_t(len(data)))

	C.omt_send(s.send, s.mf)
	return 0
}

type FrameType int

const (
	None     FrameType = iota
	Metadata           = iota
	Video              = iota
	Audio              = iota
)

type Quality int32

const (
	QualityDefault Quality = C.OMTQuality_Default
	QualityLow             = C.OMTQuality_Low
	QualityMedium          = C.OMTQuality_Medium
	QualityHigh            = C.OMTQuality_High
)

type Codec int

const (
	CodecVMX1 Codec = C.OMTCodec_VMX1
	CodecFPA1       = C.OMTCodec_FPA1
	CodecBGRA       = C.OMTCodec_BGRA
	CodecUYVY       = C.OMTCodec_UYVY
	CodecYUY2       = C.OMTCodec_YUY2
	CodecNV12       = C.OMTCodec_NV12
	CodecYV12       = C.OMTCodec_YV12
	CodecUYVA       = C.OMTCodec_UYVA
	CodecP216       = C.OMTCodec_P216
	CodecPA16       = C.OMTCodec_PA16
)

type ColorSpace int

const (
	ColorSpaceUndefined ColorSpace = C.OMTColorSpace_Undefined
	ColorSpaceBT601     ColorSpace = C.OMTColorSpace_BT601
	ColorSpaceBT709     ColorSpace = C.OMTColorSpace_BT709
)

type VideoFlags int

const (
	VideoFlagsNone          VideoFlags = 0
	VideoFlagsInterlaced               = 1
	VideoFlagsAlpha                    = 2
	VideoFlagsPreMultiplied            = 4
	VideoFlagsPreview                  = 4
	VideoFlagsHighBitDepth             = 16
)

type OmtMediaFrame struct {
	Width         int
	Height        int
	Codec         Codec
	Timestamp     int64
	ColorSpace    ColorSpace
	Flags         VideoFlags
	Stride        int
	DataLength    int
	FrameRateN    int
	FrameRateD    int
	AspectRatio   float32
	FrameMetadata []byte

	// Audio properties
	SampleRate        int
	Channels          int
	SamplesPerChannel int
}
