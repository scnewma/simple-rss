package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
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
	outputPath := fs.String("output", "index.html", "path to output HTML file")
	if err := fs.Parse(argv); err != nil {
		return err
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ContextWithLogger(ctx, logger)

	feeds, fetchErr := new(Fetcher).FetchAll(ctx, cfg.Feeds)
	if fetchErr != nil {
		logger.Error("fetching feeds failed", "error", fetchErr.Error())
		// fallthrough as we probably still successfully fetched a
		// subset of the feeds
	}

	if len(feeds) == 0 {
		return fetchErr
	}

	if err := WriteHTML(*outputPath, feeds); err != nil {
		return fmt.Errorf("writing html: %w", err)
	}

	logger.Info("wrote feeds", "path", *outputPath)
	return nil
}
