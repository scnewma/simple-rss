package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/robfig/cron/v3"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, argv []string) error {
	fs := flag.NewFlagSet("simple-rss", flag.ContinueOnError)
	configPath := fs.String("config", "config.json", "path to config file")
	if err := fs.Parse(argv); err != nil {
		return err
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ContextWithLogger(ctx, logger)

	fetcher := new(Fetcher)

	refresh := func() error {
		feeds, err := fetcher.FetchAll(ctx, cfg.Feeds)
		if err != nil {
			logger.Error("fetching feeds failed", "error", err.Error())
			// fallthrough as we probably still successfully fetched a
			// subset of the feeds
		}

		if len(feeds) == 0 {
			return err
		}

		if err := WriteHTML(cfg.OutputPath, feeds); err != nil {
			return fmt.Errorf("writing html: %w", err)
		}

		logger.Info("refreshed feeds", "path", cfg.OutputPath)
		return nil
	}

	// TODO: if OutputPath does not exist, poll immediately
	if !fileExists(cfg.OutputPath) {
		if err := refresh(); err != nil {
			logger.Error("refreshing feeds failed", "error", err)
		}
	}

	scheduler := cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	if _, err := scheduler.AddFunc(cfg.PollCron, func() {
		if err := refresh(); err != nil {
			logger.Error("refreshing feeds failed", "error", err)
		}
	}); err != nil {
		return fmt.Errorf("schedule poll: %w", err)
	}
	scheduler.Start()
	<-ctx.Done()
	logger.Info("shutting down... waiting for running jobs to complete")
	<-scheduler.Stop().Done()
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
