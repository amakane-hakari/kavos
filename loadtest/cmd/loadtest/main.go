// Package main は 負荷試験ツールのエントリーポイントを提供します。
package main

import (
	"fmt"
	"os"

	"github.com/amakane-hakari/kavos/loadtest/attacker"
	"github.com/amakane-hakari/kavos/loadtest/config"
	"github.com/amakane-hakari/kavos/loadtest/scenario"
)

func main() {
	cfg := config.Load()

	fmt.Printf("[INFO] base-url=%s rate=%d duration=%s read-ratio=%.2f keys=%d value-size=%d ttl-ratio=%.2f ttl-ms=%d read-only=%v\n",
		cfg.BaseURL, cfg.Rate, cfg.Duration, cfg.ReadRatio, cfg.Keys, cfg.ValueSize, cfg.TTLRatio, cfg.TTLMillis, cfg.DisablePUT)

	gen := scenario.NewGenerator(
		cfg.BaseURL,
		cfg.Keys,
		cfg.ReadRatio,
		cfg.ValueSize,
		cfg.TTLRatio,
		cfg.TTLMillis,
		cfg.DisablePUT,
	)

	r := attacker.Runner{
		Rate:     cfg.Rate,
		Duration: cfg.Duration,
		Timeout:  cfg.Timeout,
		Name:     cfg.Name,
		Output:   cfg.Output,
	}

	if _, err := r.Run(gen.Targeter()); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
