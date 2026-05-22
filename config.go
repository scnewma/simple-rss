package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"
)

type Config struct {
	Feeds  []string      `json:"feeds"`
	Groups []GroupConfig `json:"groups"`
}

type GroupConfig struct {
	Title  string   `json:"title"`
	MaxAge Duration `json:"maxAge"`
}

func defaultGroups() []GroupConfig {
	return []GroupConfig{
		{Title: "Today", MaxAge: Duration(24 * time.Hour)},
		{Title: "Last 7 Days", MaxAge: Duration(7 * 24 * time.Hour)},
		{Title: "Last 30 Days", MaxAge: Duration(30 * 24 * time.Hour)},
		{Title: "Older"},
	}
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

	if len(cfg.Groups) == 0 {
		cfg.Groups = defaultGroups()
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

	for _, group := range cfg.Groups {
		if group.Title == "" {
			return fmt.Errorf("group title must not be empty")
		}
		if group.MaxAge.Duration() < 0 {
			return fmt.Errorf("group %q maxAge must not be negative", group.Title)
		}
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
