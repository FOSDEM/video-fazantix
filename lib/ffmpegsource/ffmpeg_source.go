package ffmpegsource

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
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

	f.frames.IsReady = true
	return true
}

func (f *FFmpegSource) setupCmd() error {
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
		frame := f.frames.GetBlankFrame()
		err := encdec.PrepareYUYV422p(frame)
		if err != nil {
			f.log("Could not prepare YUV422 buffer: %s", err)
			return
		}
		_, err = io.ReadFull(f.stdout, frame.Data)
		if err != nil {
			f.log("could not read from ffmpeg's output: %s", err)
			return
		}

		f.frames.SendFrame(frame)
	}
}

func (f *FFmpegSource) Frames() *layer.FrameForwarder {
	return &f.frames
}

func (f *FFmpegSource) log(msg string, args ...interface{}) {
	f.Frames().Log(msg, args...)
}
