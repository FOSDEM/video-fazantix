package ffmpegsink

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"

	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/layer"
)

type FFmpegSink struct {
	name     string
	shellCmd string
	cmd      *exec.Cmd
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	stdin    io.WriteCloser
	frames   layer.FrameForwarder
}

func New(name string, cfg *config.FFmpegSinkCfg) *FFmpegSink {
	f := &FFmpegSink{shellCmd: cfg.Cmd}
	f.name = name
	f.frames.Init(encdec.RGBFrames, []uint8{}, cfg.W, cfg.H)
	return f
}

func (f *FFmpegSink) Name() string {
	return f.name
}

func (f *FFmpegSink) Start() bool {
	var err error

	err = f.setupCmd()
	if err != nil {
		log.Printf("could not setup ffmpeg command: %s", err)
		return false
	}

	go f.runFFmpeg()
	go f.processStdout()
	go f.processStderr()

	f.frames.IsReady = true
	return true
}

func (f *FFmpegSink) setupCmd() error {
	f.cmd = exec.Command("bash", "-c", f.shellCmd)
	var err error
	f.stdin, err = f.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("could not get ffmpeg stdin: %s\n", err)
	}
	f.stdout, err = f.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not get ffmpeg stdout: %s\n", err)
	}
	f.stderr, err = f.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("could not get ffmpeg stderr: %s\n", err)
	}
	return nil
}

func (f *FFmpegSink) runFFmpeg() {
	for {
		log.Printf("starting ffmpeg")

		err := f.cmd.Run()
		if err != nil {
			log.Printf("ffmpeg error: %s\n", err)
		}

		log.Printf("ffmpeg died")
		err = f.setupCmd()
		if err != nil {
			log.Printf("could not setup ffmpeg command: %s\n", err)
			time.Sleep(5 * time.Second)
			continue
		}
		time.Sleep(1 * time.Second)
	}
}

func (f *FFmpegSink) processStderr() {
	scanner := bufio.NewScanner(f.stderr)
	for scanner.Scan() {
		log.Printf("[ffmpeg] %s", scanner.Text())
	}
}

func (f *FFmpegSink) processStdout() {
	scanner := bufio.NewScanner(f.stdout)
	for scanner.Scan() {
		log.Printf("[ffmpeg] %s", scanner.Text())
	}
}

func (f *FFmpegSink) processStdin() {
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

func (f *FFmpegSink) Frames() *layer.FrameForwarder {
	return &f.frames
}
