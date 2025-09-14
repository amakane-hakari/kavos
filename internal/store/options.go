package store

import (
	"time"

	"github.com/amakane-hakari/kavos/internal/metrics"
)

type logLike interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Config はストアの設定を表します。
type Config struct {
	Shards             int           // 2 の冪推奨。0/未指定なら 16
	CleanupInterval    time.Duration // 0 で無効
	Logger             logLike
	Metrics            metrics.Interface
	EnableShardPadding bool // シャードのパディングを有効にする
}

// Option はストアのオプションを設定する関数です。
type Option func(*Config)

// WithLogger はストアのロガーを設定するオプションです。
func WithLogger(l logLike) Option {
	return func(c *Config) { c.Logger = l }
}

// WithMetrics はストアのメトリクスを設定するオプションです。
func WithMetrics(m metrics.Interface) Option {
	return func(c *Config) { c.Metrics = m }
}

// WithShards はストアのシャード数を設定するオプションです。
func WithShards(n int) Option {
	return func(c *Config) { c.Shards = n }
}

// WithCleanupInterval はストアのクリーンアップ間隔を設定するオプションです。
func WithCleanupInterval(d time.Duration) Option {
	return func(c *Config) { c.CleanupInterval = d }
}

// WithShardPadding はストアのシャードパディングを有効にするオプションです。
func WithShardPadding() Option {
	return func(c *Config) { c.EnableShardPadding = true }
}
