package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	PollCron     string   `json:"pollCron"`
	DisplayDays  int      `json:"displayDays"`
	ListenAddr   string   `json:"listenAddr"`
	DatabasePath string   `json:"databasePath"`
	Feeds        []string `json:"feeds"`
}

func Default() Config {
	cfg := Config{}
	applyDefaults(&cfg)
	return cfg
}

func Load(path string) (Config, error) {
	cfg := Default()

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

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.PollCron == "" {
		cfg.PollCron = "0 0 0 * * *"
	}
	if cfg.DisplayDays == 0 {
		cfg.DisplayDays = 90
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = "simple-rss.db"
	}
}

func validate(cfg Config) error {
	if cfg.PollCron == "" {
		return fmt.Errorf("pollCron is required")
	}
	if cfg.DisplayDays <= 0 {
		return fmt.Errorf("displayDays must be positive")
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
