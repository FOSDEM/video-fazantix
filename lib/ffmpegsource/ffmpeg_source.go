package ffmpegsource

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"time"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
)

type FFmpegSource struct {
	shellCmd string
	cmd      *exec.Cmd
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	frames   layer.FrameForwarder
}

func New(name string, cfg *config.FFmpegSourceCfg, alloc encdec.FrameAllocator) *FFmpegSource {
	f := &FFmpegSource{shellCmd: cfg.Cmd}
	f.frames.Init(
		name,
		&encdec.FrameInfo{
			FrameType: encdec.YUV422Frames,
			PixFmt:    []uint8{},
			FrameCfg:  cfg.FrameCfg,
		},
		alloc,
	)
	return f
}

func (f *FFmpegSource) Start() bool {
	err := f.setupCmd()
	if err != nil {
		f.log("could not setup ffmpeg command: %s", err)
		return false
	}

	go f.runFFmpeg()
	go f.processStdout()
	go f.processStderr()

	return true
}

func (f *FFmpegSource) setupCmd() error {
	f.cmd = exec.Command("bash", "-c", f.shellCmd)
	f.cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
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

func (f *FFmpegSource) runFFmpeg() {
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

func (f *FFmpegSource) processStderr() {
	scanner := bufio.NewScanner(f.stderr)
	for scanner.Scan() {
		f.log("[ffmpeg] %s", scanner.Text())
	}
}

func (f *FFmpegSource) processStdout() {
	for {
		frame := f.frames.GetFrameForWriting()
		if frame == nil {
			panic("framedropping for ffmpeg sources not yet implemented")
			// TODO: here we should discard the exact amount of data from
			// ffmpeg's stdout and continue the loop
		}
		err := encdec.PrepareYUYV422p(frame)
		if err != nil {
			f.log("Could not prepare YUV422 buffer: %s", err)
			f.frames.FailedWriting(frame)
			return
		}
		_, err = io.ReadFull(f.stdout, frame.Data)
		if err != nil {
			f.log("could not read from ffmpeg's output: %s", err)
			f.frames.FailedWriting(frame)
			return
		}

		f.frames.FinishedWriting(frame)
	}
}

func (f *FFmpegSource) Frames() *layer.FrameForwarder {
	return &f.frames
}

func (f *FFmpegSource) log(msg string, args ...interface{}) {
	f.Frames().Log(msg, args...)
}
