package attacker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

// ResultSummary は 負荷試験の結果概要を表します。
type ResultSummary struct {
	Requests    uint64                `json:"requests"`
	Rate        float64               `json:"rate_req_per_sec"`
	Success     float64               `json:"success_ratio"`
	Throughput  float64               `json:"throughput_bytes_per_sec"`
	Latencies   vegeta.LatencyMetrics `json:"latencies"`
	StatusCodes map[string]int        `json:"status_codes"`
	Errors      []string              `json:"errors"`
	Duration    time.Duration         `json:"duration"`
}

// Runner は 負荷試験を実行するための構造体です。
type Runner struct {
	Rate     int
	Duration time.Duration
	Timeout  time.Duration
	Name     string
	Output   string
}

// Run は 指定されたターゲッターを使用して負荷試験を実行し、結果の概要を返します。
func (r *Runner) Run(targeter vegeta.Targeter) (*ResultSummary, error) {
	rate := vegeta.Rate{Freq: r.Rate, Per: time.Second}
	att := vegeta.NewAttacker(vegeta.Timeout(r.Timeout))

	results := att.Attack(targeter, rate, r.Duration, r.Name)

	var buf bytes.Buffer
	enc := vegeta.NewEncoder(&buf)

	var metrics vegeta.Metrics
	for res := range results {
		metrics.Add(res)
		if err := enc.Encode(res); err != nil {
			return nil, fmt.Errorf("encode: %w", err)
		}
	}
	metrics.Close()

	if err := os.WriteFile(r.Output, buf.Bytes(), 0o644); err != nil {
		return nil, fmt.Errorf("write results: %w", err)
	}

	summary := &ResultSummary{
		Requests:    metrics.Requests,
		Rate:        metrics.Rate,
		Success:     metrics.Success,
		Throughput:  metrics.Throughput,
		Latencies:   metrics.Latencies,
		StatusCodes: metrics.StatusCodes,
		Errors:      metrics.Errors,
		Duration:    metrics.Duration,
	}

	reqJSON, _ := json.MarshalIndent(summary, "", " ")
	fmt.Printf("\n=== Summary(JSON) ===\n%s\n", string(reqJSON))

	return summary, nil
}
