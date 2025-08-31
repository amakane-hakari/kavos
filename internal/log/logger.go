package log

import (
	"log/slog"
	"os"
	"strings"
)

// Logger は KVSのロガーインターフェースです。
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Slog は slog.Logger のラッパーです。
type Slog struct {
	l *slog.Logger
}

// New は新しい Slog を作成します。
func New() *Slog {
	level := slog.LevelInfo
	if strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug") {
		level = slog.LevelDebug
	}
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return &Slog{l: slog.New(h)}
}

// Debug はデバッグレベルのログを出力します。
func (s *Slog) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }

// Info は情報レベルのログを出力します。
func (s *Slog) Info(msg string, args ...any) { s.l.Info(msg, args...) }

// Error はエラーレベルのログを出力します。
func (s *Slog) Error(msg string, args ...any) { s.l.Error(msg, args...) }
