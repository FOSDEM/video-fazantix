package ffmpegsource

import (
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/fosdem/vidmix/encdec"
	"github.com/fosdem/vidmix/layer"
)

type FFmpegSource struct {
	shellCmd string
	cmd      *exec.Cmd
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	frames   layer.FrameForwarder
	isReady  bool
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

	f.frames.Init()
	f.frames.FrameType = layer.YUV422Frames

	go f.runFFmpeg()
	go f.processStdout()
	go f.processStderr()

	f.isReady = true
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
	fe, err := os.Create("/tmp/ffmpeg_stderr")
	if err != nil {
		log.Printf("[ffmpeg] could not open stderr file\n")
		return
	}
	defer fe.Close()

	io.Copy(fe, f.stderr)
}

func (f *FFmpegSource) processStdout() {
	frameSize := f.Width() * f.Height() * 2 // bytes for yuyv422 frames
	buf := make([]byte, frameSize)
	for {
		_, err := io.ReadFull(f.stdout, buf)
		if err != nil {
			log.Printf("could not read from ffmpeg's output: %s\n", err)
			return
		}

		img, err := encdec.DecodeYUYV422(buf, f.frames.GetBlankYUV422Frame(f.Width(), f.Height()))
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

func (f *FFmpegSource) Width() int {
	return 1920
}

func (f *FFmpegSource) Height() int {
	return 1080
}

func (f *FFmpegSource) IsReady() bool {
	return f.isReady
}

func (f *FFmpegSource) IsStill() bool {
	return false
}
