//go:build omt

package libomt

/*
#cgo pkg-config: libomt
#include "libomt.h"
#include "string.h"
#include "malloc.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type OmtReceive struct {
	recv *C.omt_receive_t
}

type PreferredVideoFormat int

const (
	PreferredVideoFormatUYVY                   PreferredVideoFormat = C.OMTPreferredVideoFormat_UYVY
	PreferredVideoFormatUYVYorBGRA                                  = C.OMTPreferredVideoFormat_UYVYorBGRA
	PreferredVideoFormatBGRA                                        = C.OMTPreferredVideoFormat_BGRA
	PreferredVideoFormatUYVYorUYVA                                  = C.OMTPreferredVideoFormat_UYVYorUYVA
	PreferredVideoFormatUYVYorUYVAorP215orPA16                      = C.OMTPreferredVideoFormat_UYVYorUYVAorP216orPA16
	PreferredVideoFormatP216                                        = C.OMTPreferredVideoFormat_P216
)

type ReceiveFlags int

const (
	ReceiveFlagsNone              ReceiveFlags = C.OMTReceiveFlags_None
	ReceiveFlagsPreview                        = C.OMTReceiveFlags_Preview
	ReceiveFlagsIncludeCompressed              = C.OMTReceiveFlags_IncludeCompressed
	ReceiveFlagsCompressedOnly                 = C.OMTReceiveFlags_CompressedOnly
)

func OmtReceiveCreate(name string, frameTypes FrameType, preferredFormat PreferredVideoFormat, flags ReceiveFlags) (*OmtReceive, error) {
	result := &OmtReceive{}
	result.recv = C.omt_receive_create(C.CString(name), C.OMTFrameType(frameTypes), C.OMTPreferredVideoFormat(preferredFormat), C.OMTReceiveFlags(flags))
	if result.recv == nil {
		return nil, fmt.Errorf("failed to create OmtReceive")
	}
	return result, nil
}

func (r *OmtReceive) Receive(frameTypes FrameType, timeoutMilliseconds int, data []byte) *OmtMediaFrame {
	mf := C.omt_receive(r.recv, C.OMTFrameType(frameTypes), C.int(timeoutMilliseconds))
	if mf == nil {
		return nil
	}
	res := &OmtMediaFrame{
		Width:             int(mf.Width),
		Height:            int(mf.Height),
		Codec:             Codec(mf.Codec),
		Timestamp:         int64(mf.Timestamp),
		ColorSpace:        ColorSpace(mf.ColorSpace),
		Flags:             VideoFlags(mf.Flags),
		Stride:            int(mf.Stride),
		DataLength:        int(mf.DataLength),
		FrameRateN:        int(mf.FrameRateN),
		FrameRateD:        int(mf.FrameRateD),
		AspectRatio:       float32(mf.AspectRatio),
		FrameMetadata:     nil,
		SampleRate:        0,
		Channels:          0,
		SamplesPerChannel: 0,
	}
	gb := C.GoBytes(mf.Data, C.int(res.DataLength))
	copy(data, gb)
	return res
}

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

func (s *OmtSend) SetSenderInformation(info *OmtSenderInfo) {
	si := C.OMTSenderInfo{}
	for i, c := range info.ProductName {
		si.ProductName[i] = C.char(c)
	}
	si.ProductName[len(info.ProductName)] = 0
	for i, c := range info.Manufacturer {
		si.Manufacturer[i] = C.char(c)
	}
	si.Manufacturer[len(info.Manufacturer)] = 0
	for i, c := range info.Version {
		si.Version[i] = C.char(c)
	}
	si.Version[len(info.Version)] = 0

	C.omt_send_setsenderinformation(s.send, &si)
}

func (s *OmtSend) AddConnectionMetadata(data string) {
	C.omt_send_addconnectionmetadata(s.send, C.CString(data))
}

func (s *OmtSend) ClearConnectionMetadata() {
	C.omt_send_clearconnectionmetadata(s.send)
}

func (s *OmtSend) SetRedirect(newAddress string) {
	C.omt_send_setredirect(s.send, C.CString(newAddress))
}

func (s *OmtSend) GetAddress() string {
	buffer := (*C.char)(C.malloc(1024))
	C.omt_send_getaddress(s.send, buffer, 1024)
	return C.GoString(buffer)
}

func (s *OmtSend) Close() {
	C.omt_send_destroy(s.send)
}

func (s *OmtSend) Connections() int {
	return int(C.omt_send_connections(s.send))
}

func (s *OmtSend) Receive(timeoutMilliseconds int) *OmtMediaFrame {
	frame := C.omt_send_receive(s.send, C.int(timeoutMilliseconds))
	if frame == nil {
		return nil
	}
	result := &OmtMediaFrame{}
	// TODO: Decode
	return result
}

type Tally struct {
	Preview int
	Program int
}

func (s *OmtSend) GetTally(timeoutMilliseconds int) *Tally {
	ts := C.OMTTally{}
	ret := C.omt_send_gettally(s.send, C.int(timeoutMilliseconds), &ts)
	if ret == 0 {
		return nil
	}
	return &Tally{
		Preview: int(ts.preview),
		Program: int(ts.program),
	}
}

func (s *OmtSend) GetVideoStatistics() OmtStatistics {
	temp := C.OMTStatistics{}
	C.omt_send_getvideostatistics(s.send, &temp)
	return OmtStatistics{
		BytesSent:              int64(temp.BytesSent),
		BytesReceived:          int64(temp.BytesReceived),
		BytesSentSinceLast:     int64(temp.BytesSentSinceLast),
		BytesReceivedSinceLast: int64(temp.BytesReceivedSinceLast),
		Frames:                 int64(temp.Frames),
		FramesSinceLast:        int64(temp.FramesSinceLast),
		FramesDropped:          int64(temp.FramesDropped),
		CodecTime:              int64(temp.CodecTime),
		CodecTimeSinceLast:     int64(temp.CodecTimeSinceLast),
	}
}

type FrameType int

const (
	None     FrameType = C.OMTFrameType_None
	Metadata           = C.OMTFrameType_Metadata
	Video              = C.OMTFrameType_Video
	Audio              = C.OMTFrameType_Audio
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
	Data          []byte
	FrameRateN    int
	FrameRateD    int
	AspectRatio   float32
	FrameMetadata []byte

	// Audio properties
	SampleRate        int
	Channels          int
	SamplesPerChannel int
}

type OmtSenderInfo struct {
	ProductName  string
	Manufacturer string
	Version      string
}

type OmtStatistics struct {
	BytesSent              int64
	BytesReceived          int64
	BytesSentSinceLast     int64
	BytesReceivedSinceLast int64
	Frames                 int64
	FramesSinceLast        int64
	FramesDropped          int64
	CodecTime              int64
	CodecTimeSinceLast     int64
}
