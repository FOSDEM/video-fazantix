package ffmpegsink

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"syscall"
	"time"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
)

type FFmpegSink struct {
	shellCmd string
	cmd      *exec.Cmd
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	stdin    io.WriteCloser
	frames   layer.FrameForwarder
}

func New(name string, cfg *config.FFmpegSinkCfg, frameCfg *encdec.FrameCfg, alloc encdec.FrameAllocator) *FFmpegSink {
	f := &FFmpegSink{shellCmd: cfg.Cmd}
	f.frames.Init(
		name,
		&encdec.FrameInfo{
			FrameType: encdec.RGBAFrames,
			FrameCfg:  *frameCfg,
		},
		alloc,
	)
	return f
}

func (f *FFmpegSink) Start() bool {
	err := f.setupCmd()
	if err != nil {
		f.Frames().Error("could not setup ffmpeg command: %s", err)
		return false
	}

	go f.runFFmpeg()
	go f.processStdout()
	go f.processStderr()
	go f.processStdin()

	return true
}

func (f *FFmpegSink) setupCmd() error {
	f.cmd = exec.Command("bash", "-c", f.shellCmd)
	f.cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
	var err error
	f.stdin, err = f.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("could not get ffmpeg stdin: %s", err)
	}
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
		f.Frames().Debug("starting ffmpeg")

		err := f.cmd.Run()
		if err != nil {
			f.Frames().Error("ffmpeg error: %s", err)
		}

		f.Frames().Error("ffmpeg died")
		err = f.setupCmd()
		if err != nil {
			f.Frames().Error("could not setup ffmpeg command: %s", err)
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
		// Here we use GetAnyFrameForReading instead of GetFreshFrameForReading
		// because we want to feed duplicate frames to ffmpeg if it consumes
		// frames faster than our render loop
		frame := f.Frames().GetAnyFrameForReading()
		if frame == nil {
			continue
		}

		_, err := f.stdin.Write(frame.Data)
		f.Frames().FinishedReading(frame)
		if err != nil {
			f.Frames().Error("Could not write to ffmpeg stdin")
			return
		}
	}
}

func (f *FFmpegSink) Frames() *layer.FrameForwarder {
	return &f.frames
}

func (f *FFmpegSink) log(msg string, args ...interface{}) {
	f.Frames().Log(msg, args...)
}
