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
	shellCmd string
	cmd      *exec.Cmd
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	stdin    io.WriteCloser
	frames   layer.FrameForwarder
}

func New(name string, cfg *config.FFmpegSinkCfg) *FFmpegSink {
	f := &FFmpegSink{shellCmd: cfg.Cmd}
	f.frames.Init(name, encdec.RGBFrames, []uint8{}, cfg.W, cfg.H)
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
	go f.processStdin()

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
	scanner := bufio.NewScanner(f.stdout)
	for scanner.Scan() {
		log.Printf("[ffmpeg] %s", scanner.Text())
	}
}

func (f *FFmpegSink) processStdin() {
	for {
		frame := f.Frames().LastFrame
		if frame != nil {
			_, err := f.stdin.Write(frame.Data)
			if err != nil {
				f.log("Could not write to ffmpeg stdin")
				return
			}
		}
	}
}

func (f *FFmpegSink) Frames() *layer.FrameForwarder {
	return &f.frames
}

func (f *FFmpegSink) log(msg string, args ...interface{}) {
	f.Frames().Log(msg, args...)
}
