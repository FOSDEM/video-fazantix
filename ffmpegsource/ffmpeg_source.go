package ffmpegsource

import (
	"image"
	"io"
	"log"
	"os/exec"
	"bufio"

	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/layer"
)

type FFmpegSource struct {
	shellCmd string
	cmd      *exec.Cmd
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	frames   layer.FrameForwarder
}

func New(shellCmd string) *FFmpegSource {
	return &FFmpegSource{shellCmd: shellCmd}
}

func (f *FFmpegSource) Start() bool {
	var err error

	f.cmd = exec.Command("bash", "-c", f.shellCmd)
	f.stdout, err = f.cmd.StdoutPipe()
	if err != nil {
		log.Printf("could not get ffmpeg stdout: %s\n", err)
		return false
	}
	f.stderr, err = f.cmd.StderrPipe()
	if err != nil {
		log.Printf("could not get ffmpeg stderr: %s\n", err)
		return false
	}

	f.frames.Init(layer.YUV422Frames, []uint8{}, 1920, 1080)

	go f.runFFmpeg()
	go f.processStdout()
	go f.processStderr()

	f.frames.IsReady = true
	return true
}

func (f *FFmpegSource) runFFmpeg() {
	err := f.cmd.Run()
	if err != nil {
		log.Printf("ffmpeg error: %s\n", err)
	}

	log.Printf("ffmpeg died")
}

func (f *FFmpegSource) processStderr() {
	scanner := bufio.NewScanner(f.stderr)
	for scanner.Scan() {
		log.Printf("[ffmpeg] %s", scanner.Text())
	}
}

func (f *FFmpegSource) processStdout() {
	frameSize := f.frames.Width * f.frames.Height * 2 // bytes for yuyv422 frames
	buf := make([]byte, frameSize)
	for {
		_, err := io.ReadFull(f.stdout, buf)
		if err != nil {
			log.Printf("could not read from ffmpeg's output: %s\n", err)
			return
		}

		imgg := f.frames.GetBlankFrame()
		img := imgg.(*image.YCbCr)
		err = encdec.DecodeYUYV422(buf, img)
		if err != nil {
			log.Printf("could not decode frame: %s\n", err)
			continue
		}
		f.frames.SendFrame(img)
	}
}

func (f *FFmpegSource) Frames() *layer.FrameForwarder {
	return &f.frames
}
