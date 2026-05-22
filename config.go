package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	Feeds []string `json:"feeds"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validateConfig(cfg Config) error {
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
