package ffmpegsource

import (
	"bufio"
	"io"
	"log"
	"os/exec"
	"time"

	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/layer"
)

type FFmpegSource struct {
	name     string
	shellCmd string
	cmd      *exec.Cmd
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	frames   layer.FrameForwarder
}

func New(name string, cfg *config.FFmpegSourceCfg) *FFmpegSource {
	f := &FFmpegSource{shellCmd: cfg.Cmd}
	f.name = name
	f.frames.Init(encdec.YUV422Frames, []uint8{}, cfg.W, cfg.H)
	return f
}

func (f *FFmpegSource) Name() string {
	return f.name
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

	go f.runFFmpeg()
	go f.processStdout()
	go f.processStderr()

	f.frames.IsReady = true
	return true
}

func (f *FFmpegSource) runFFmpeg() {
	for {
		log.Printf("starting ffmpeg")
		err := f.cmd.Run()
		if err != nil {
			log.Printf("ffmpeg error: %s\n", err)
		}

		log.Printf("ffmpeg died")
		time.Sleep(1 * time.Second)
	}
}

func (f *FFmpegSource) processStderr() {
	scanner := bufio.NewScanner(f.stderr)
	for scanner.Scan() {
		log.Printf("[ffmpeg] %s", scanner.Text())
	}
}

func (f *FFmpegSource) processStdout() {
	for {
		frame := f.frames.GetBlankFrame()
		encdec.PrepareYUYV422p(frame)
		_, err := io.ReadFull(f.stdout, frame.Data)
		if err != nil {
			log.Printf("could not read from ffmpeg's output: %s\n", err)
			return
		}

		f.frames.SendFrame(frame)
	}
}

func (f *FFmpegSource) Frames() *layer.FrameForwarder {
	return &f.frames
}
