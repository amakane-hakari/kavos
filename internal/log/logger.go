package log

import (
	"log/slog"
	"os"
	"strings"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type Slog struct {
	l *slog.Logger
}

func New() *Slog {
	level := slog.LevelInfo
	if strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug") {
		level = slog.LevelDebug
	}
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return &Slog{l: slog.New(h)}
}

func (s *Slog) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }
func (s *Slog) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s *Slog) Error(msg string, args ...any) { s.l.Error(msg, args...) }