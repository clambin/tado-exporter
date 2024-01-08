package testutil

import (
	"bytes"
	"log/slog"
)

func NewBufferLogger(buffer *bytes.Buffer) *slog.Logger {
	opts := slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}}
	return slog.New(slog.NewTextHandler(buffer, &opts))
}
