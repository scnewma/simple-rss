package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/robfig/cron/v3"

	"simple-rss/config"
	"simple-rss/poller"
	"simple-rss/storage"
	"simple-rss/web"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "config.json", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	store, err := storage.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer store.Close()

	feedPoller := poller.New(cfg.Feeds, store, logger)
	scheduler := cron.New(cron.WithSeconds())
	if _, err := scheduler.AddFunc(cfg.PollCron, func() {
		feedPoller.Poll(context.Background())
	}); err != nil {
		return fmt.Errorf("schedule poll: %w", err)
	}
	scheduler.Start()
	defer scheduler.Stop()

	logger.Info("starting server", "addr", cfg.ListenAddr)

	return http.ListenAndServe(cfg.ListenAddr, web.Handler(store, cfg.DisplayDays, logger))
}
