package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"simple-rss/poller"
	"simple-rss/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	out := flag.String("out", "example.db", "output SQLite database path")
	flag.Parse()

	if err := os.Remove(*out); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing db: %w", err)
	}

	feedXML, err := os.ReadFile(filepath.Join("poller", "testdata", "matklad-feed.xml"))
	if err != nil {
		return fmt.Errorf("read feed fixture: %w", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write(feedXML)
	}))
	defer server.Close()

	store, err := storage.Open(*out)
	if err != nil {
		return err
	}
	defer store.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	summary := poller.New([]string{server.URL}, store, logger).Poll(context.Background())
	if len(summary.FailedFeeds) > 0 {
		return fmt.Errorf("poll failed: %v", summary.FailedFeeds)
	}

	logger.Info("example database created", "path", *out, "articles", summary.ItemsInserted)
	return nil
}
