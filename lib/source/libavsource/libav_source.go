package libavsource

import (
	"errors"
	"fmt"
	"log"

	"github.com/asticode/go-astiav"
	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
)

type LibavSource struct {
	Format string
	frames layer.FrameForwarder

	Path                   string
	hardwareDecode         bool
	hardwareDeviceTypeName string
	hardwareDeviceName     string
	decoderCodecName       string

	inputFormat           *astiav.InputFormat
	hardwareDeviceType    astiav.HardwareDeviceType
	packet                *astiav.Packet
	hwframe               *astiav.Frame
	inputFormatContext    *astiav.FormatContext
	inputStream           *astiav.Stream
	decoderCodec          *astiav.Codec
	hardwareDeviceContext *astiav.HardwareDeviceContext

	width          int
	height         int
	alloc          encdec.FrameAllocator
	decoderContext *astiav.CodecContext
}

func New(name string, cfg *config.LibavSourceCfg, alloc encdec.FrameAllocator) *LibavSource {
	s := &LibavSource{}
	s.alloc = alloc
	s.Path = cfg.Path
	s.Frames().Name = name
	s.frames.InitLogging()

	s.hardwareDecode = cfg.HardwareDeviceType != ""
	if s.hardwareDecode {
		s.hardwareDeviceType = astiav.FindHardwareDeviceTypeByName(cfg.HardwareDeviceType)

		if s.hardwareDeviceType == astiav.HardwareDeviceTypeNone {
			panic("Device type not found")
		}
	}
	s.hardwareDeviceTypeName = cfg.HardwareDeviceType
	s.decoderCodecName = cfg.DecoderCodec

	if cfg.InputFormat != "" {
		astiav.RegisterAllDevices()
		s.inputFormat = astiav.FindInputFormat(cfg.InputFormat)
		s.Frames().Debug("Input format %s: %s", cfg.InputFormat, s.inputFormat)
	}

	return s
}

func (s *LibavSource) Frames() *layer.FrameForwarder {
	return &s.frames
}

func (s *LibavSource) Start() bool {

	s.packet = astiav.AllocPacket()
	//defer s.packet.Free()

	s.hwframe = astiav.AllocFrame()
	//defer s.hwframe.Free()

	inputFormatContext := astiav.AllocFormatContext()
	if inputFormatContext == nil {
		s.Frames().Error("Unable to allocate input format context")
		return false
	}
	s.inputFormatContext = inputFormatContext
	//defer s.inputFormatContext.Free()

	inputOptions := astiav.NewDictionary()
	inputOptions.Set("input_format", "h264", 0)

	err := inputFormatContext.OpenInput(s.Path, s.inputFormat, inputOptions)
	if err != nil {
		s.Frames().Error("open: %s\n", err)
		return false
	}
	//defer s.inputFormatContext.CloseInput()

	err = inputFormatContext.FindStreamInfo(nil)
	if err != nil {
		s.Frames().Error("find-stream-info: %s\n", err)
		return false
	}

	for _, stream := range inputFormatContext.Streams() {
		s.Frames().Log("Input stream: %s %dx%d", stream.CodecParameters().CodecID().Name(), stream.CodecParameters().Width(), stream.CodecParameters().Height())
		if stream.CodecParameters().MediaType() != astiav.MediaTypeVideo {
			continue
		}
		s.inputStream = stream
		s.width = stream.CodecParameters().Width()
		s.height = stream.CodecParameters().Height()
		if s.decoderCodecName == "" {
			if s.hardwareDecode {
				s.decoderCodecName = fmt.Sprintf("%s_%s", stream.CodecParameters().CodecID().Name(), s.hardwareDeviceTypeName)
			} else {
				s.decoderCodecName = stream.CodecParameters().CodecID().Name()
			}
		}
		s.Frames().Debug("Decoder codec: %s", s.decoderCodecName)
		s.decoderCodec = astiav.FindDecoderByName(s.decoderCodecName)
		if s.decoderCodec == nil {
			s.Frames().Error("Decoder codec not found: %s", s.decoderCodecName)
			return false
		}
		break
	}
	decCodecContext := astiav.AllocCodecContext(s.decoderCodec)
	if decCodecContext == nil {
		s.Frames().Error("Unable to allocate decoder codec: %s", s.decoderCodecName)
		return false
	}
	//defer decCodecContext.Free()

	err = s.inputStream.CodecParameters().ToCodecContext(decCodecContext)
	if err != nil {
		s.Frames().Error("Unable to update codec context: %s", err)
		return false
	}

	if s.hardwareDecode {
		s.hardwareDeviceContext, err = astiav.CreateHardwareDeviceContext(s.hardwareDeviceType, s.hardwareDeviceName, nil, 0)
		if err != nil {
			s.Frames().Error("Unable to create hardware context: %s", err)
			return false
		}
		defer s.hardwareDeviceContext.Free()

		hardwareFrameConstraints := s.hardwareDeviceContext.HardwareFramesConstraints()
		if hardwareFrameConstraints == nil {
			s.Frames().Error("Hardware frame constraints is null")
			return false
		}
		defer hardwareFrameConstraints.Free()

		validHardwarePixelFormats := hardwareFrameConstraints.ValidHardwarePixelFormats()
		if len(validHardwarePixelFormats) == 0 {
			s.Frames().Error("No valid hardware pixel formats")
			return false
		}

		log.Println("Formats:", validHardwarePixelFormats)
	}

	pf := decCodecContext.PixelFormat()
	log.Println(decCodecContext.String())
	log.Println(pf.Descriptor().Name())
	err = decCodecContext.Open(s.decoderCodec, nil)
	if err != nil {
		s.Frames().Error("Unable to start decoder codec: %s", err)
		return false
	}
	s.decoderContext = decCodecContext

	var frametype encdec.FrameType
	switch decCodecContext.PixelFormat().Name() {
	case "yuv420p":
		// 4:2:0 planar
		frametype = encdec.YUV420Frames
	default:
		panic(decCodecContext.PixelFormat().Name())
	}

	s.frames.Init(
		s.frames.Name,
		&encdec.FrameInfo{
			FrameCfg: encdec.FrameCfg{
				Width:              s.width,
				Height:             s.height,
				NumAllocatedFrames: 2,
			},
			FrameType: frametype,
			PixFmt:    []uint8{},
		},
		s.alloc,
	)
	s.frames.IsReady = true

	go s.processFrames()
	return true
}

func (s *LibavSource) Stop() {
}

func (s *LibavSource) processFrames() {

	// Loop through source packets
	for {
		if stop := func() bool {
			// Read source packet
			err := s.inputFormatContext.ReadFrame(s.packet)
			if err != nil {
				if errors.Is(err, astiav.ErrEof) {
					log.Println("EOF")
					return true
				}
				return true
			}
			defer s.packet.Unref()

			if s.packet.StreamIndex() != s.inputStream.Index() {
				// Packet is not for the video stream
				return false
			}

			// Send packet to codec
			err = s.decoderContext.SendPacket(s.packet)
			if err != nil {
				s.Frames().Error("Unable to send packet: %s", err)
				return true
			}

			// Loop through frames in packet
			for {
				if run := func() bool {
					// Get frame from codec
					err := s.decoderContext.ReceiveFrame(s.hwframe)
					if err != nil {
						if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
							// No more frames in this packet
							return false
						}
						panic(err)
					}
					defer s.hwframe.Unref()

					frame := s.Frames().GetFrameForWriting()
					encdec.PrepareYUV420(frame)
					raw, err := s.hwframe.Data().Bytes(4)
					if err != nil {
						s.Frames().FailedWriting(frame)
						panic(err)
					}
					copy(frame.Data, raw)
					s.Frames().FinishedWriting(frame)

					return false
				}(); !run {
					break
				}
			}

			return false
		}(); stop {
			break
		}
	}
}
