package ffmpegsink

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/layer"
)

type FFmpegSink struct {
	shellCmd string
	cmd      *exec.Cmd
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	frames   layer.FrameForwarder
}

func New(name string, cfg *config.FFmpegSinkCfg) *FFmpegSink {
	f := &FFmpegSink{shellCmd: cfg.Cmd}
	f.frames.Init(name, encdec.YUV422Frames, []uint8{}, cfg.W, cfg.H)
	return f
}

func (f *FFmpegSink) Start() bool {
	var err error

	err = f.setupCmd()
	if err != nil {
		f.log("could not setup ffmpeg command: %s", err)
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
	f.stdout, err = f.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not get ffmpeg stdout: %s", err)
	}
	f.stderr, err = f.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("could not get ffmpeg stderr: %s", err)
	}
	return nil
}

func (f *FFmpegSink) runFFmpeg() {
	for {
		f.log("starting ffmpeg")

		err := f.cmd.Run()
		if err != nil {
			f.log("ffmpeg error: %s", err)
		}

		f.log("ffmpeg died")
		err = f.setupCmd()
		if err != nil {
			f.log("could not setup ffmpeg command: %s", err)
			time.Sleep(5 * time.Second)
			continue
		}
		time.Sleep(1 * time.Second)
	}
}

func (f *FFmpegSink) processStderr() {
	scanner := bufio.NewScanner(f.stderr)
	for scanner.Scan() {
		f.log("[ffmpeg] %s", scanner.Text())
	}
}

func (f *FFmpegSink) processStdout() {
	panic("wtf are you doing")
	for {
		frame := f.frames.GetBlankFrame()
		encdec.PrepareYUYV422p(frame)
		_, err := io.ReadFull(f.stdout, frame.Data)
		if err != nil {
			f.log("could not read from ffmpeg's output: %s", err)
			return
		}

		f.frames.SendFrame(frame)
	}
}

func (f *FFmpegSink) Frames() *layer.FrameForwarder {
	return &f.frames
}

func (f *FFmpegSink) log(msg string, args ...interface{}) {
	f.Frames().Log(msg, args...)
}
