package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
)

type LogHandler struct {
	subHandler  slog.Handler
	buffer      *bytes.Buffer
	bufferMutex *sync.Mutex
}

const (
	reset = "\033[0m"

	black        = 30
	red          = 31
	green        = 32
	yellow       = 33
	blue         = 34
	magenta      = 35
	cyan         = 36
	lightGray    = 37
	darkGray     = 90
	lightRed     = 91
	lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	lightCyan    = 96
	white        = 97
)

func colorize(colorCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), v, reset)
}

func (h *LogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.subHandler.Enabled(ctx, level)
}

func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{subHandler: h.subHandler.WithAttrs(attrs), buffer: h.buffer, bufferMutex: h.bufferMutex}
}

func (h *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{subHandler: h.subHandler.WithGroup(name), buffer: h.buffer, bufferMutex: h.bufferMutex}
}

func (h *LogHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String() + " "

	switch r.Level {
	case slog.LevelDebug:
		level = colorize(darkGray, level)
	case slog.LevelInfo:
		level = colorize(cyan, level)
	case slog.LevelWarn:
		level = colorize(lightYellow, level)
	case slog.LevelError:
		level = colorize(lightRed, level)
	}

	attrs, err := h.parseAttributes(ctx, r)
	if err != nil {
		return err
	}

	fmt.Print(colorize(lightGray, r.Time.Format("15:04:05.000 ")))
	fmt.Print(level)
	if attrs["module"] != nil {
		fmt.Print(colorize(lightGray, fmt.Sprintf("[%s] ", attrs["module"])))
	}
	fmt.Println(r.Message)
	return nil
}

func (h *LogHandler) parseAttributes(ctx context.Context, r slog.Record) (map[string]any, error) {
	h.bufferMutex.Lock()
	defer func() {
		h.buffer.Reset()
		h.bufferMutex.Unlock()
	}()
	if err := h.subHandler.Handle(ctx, r); err != nil {
		return nil, fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}

	var attrs map[string]any
	err := json.Unmarshal(h.buffer.Bytes(), &attrs)
	if err != nil {
		return nil, fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}
	return attrs, nil
}

func NewHandler(opts *slog.HandlerOptions) *LogHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	b := &bytes.Buffer{}
	return &LogHandler{
		buffer: b,
		subHandler: slog.NewJSONHandler(b, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: opts.ReplaceAttr,
		}),
		bufferMutex: &sync.Mutex{},
	}
}
