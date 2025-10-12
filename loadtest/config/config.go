package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type Config struct {
	BaseURL    string
	Keys       int
	ReadRatio  float64
	Rate       int
	Duration   time.Duration
	ValueSize  int
	TTLRatio   float64
	TTLMillis  int
	Output     string
	Timeout    time.Duration
	Name       string
	DisablePUT bool
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseFloatEnv(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func parseIntEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func Load() *Config {
	var c Config

	defaultBase := envOr("LT_BASE_URL", "http://localhost:8080")
	defaultKeys := parseIntEnv("LT_KEYS", 5000)
	defaultRead := parseFloatEnv("LT_READ_RATIO", 0.8)
	defaultRate := parseIntEnv("LT_RATE", 100)
	defaultDuration := envOr("LT_DURATION", "30s")
	defaultValueSize := parseIntEnv("LT_VALUE_SIZE", 128)
	defaultTTLRatio := parseFloatEnv("LT_TTL_RATIO", 0.0)
	defaultTTLMillis := parseIntEnv("LT_TTL_MILLIS", 10000)
	defaultOutput := envOr("LT_OUTPUT", "vegeta_results.bin")
	defaultTimeout := envOr("LT_TIMEOUT", "5s")
	defaultName := envOr("LT_NAME", "mixed")
	disablePUT := os.Getenv("LT_DISABLE_PUT") == "1" || os.Getenv("LT_DISABLE_PUT") == "true"

	dur, _ := time.ParseDuration(defaultDuration)
	to, _ := time.ParseDuration(defaultTimeout)

	flag.StringVar(&c.BaseURL, "base-url", defaultBase, "Base URL of the key-value store")
	flag.IntVar(&c.Keys, "keys", defaultKeys, "Number of keys to load")
	flag.Float64Var(&c.ReadRatio, "read-ratio", defaultRead, "Ratio of read operations")
	flag.IntVar(&c.Rate, "rate", defaultRate, "Rate limit for requests")
	flag.DurationVar(&c.Duration, "duration", dur, "Duration of the load test")
	flag.IntVar(&c.ValueSize, "value-size", defaultValueSize, "Size of each value")
	flag.Float64Var(&c.TTLRatio, "ttl-ratio", defaultTTLRatio, "Ratio of TTL values")
	flag.IntVar(&c.TTLMillis, "ttl-millis", defaultTTLMillis, "TTL value in milliseconds")
	flag.StringVar(&c.Output, "output", defaultOutput, "Output format")
	flag.DurationVar(&c.Timeout, "timeout", to, "Request timeout")
	flag.StringVar(&c.Name, "name", defaultName, "Name of the load test")
	flag.BoolVar(&c.DisablePUT, "disable-put", disablePUT, "Disable PUT requests")

	flag.Parse()
	return &c
}
