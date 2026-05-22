package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	OutputPath string   `json:"output_path"`
	PollCron   string   `json:"pollCron"`
	Feeds      []string `json:"feeds"`
}

func DefaultConfig() Config {
	cfg := Config{}
	applyDefaults(&cfg)
	return cfg
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	applyDefaults(&cfg)

	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	cfg.PollCron = cmp.Or(cfg.PollCron, "0 0 * * *")
	cfg.OutputPath = cmp.Or(cfg.OutputPath, "index.html")
}

func validateConfig(cfg Config) error {
	if cfg.PollCron == "" {
		return fmt.Errorf("pollCron is required")
	}
	if len(cfg.Feeds) == 0 {
		return fmt.Errorf("feeds must not be empty")
	}

	for _, raw := range cfg.Feeds {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("invalid feed URL: %q", raw)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("feed URL must use http or https: %q", raw)
		}
	}

	return nil
}
