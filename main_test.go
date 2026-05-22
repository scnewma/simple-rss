package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func TestApp(t *testing.T) {
	feedPaths, err := filepath.Glob(filepath.Join("testdata", "*"))
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	t.Cleanup(server.Close)

	feedURLs := make([]string, 0, len(feedPaths))
	for _, feedPath := range feedPaths {
		feedURLs = append(feedURLs, server.URL+"/"+filepath.Base(feedPath))
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "index.html")
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := Config{
		OutputPath: outputPath,
		PollCron:   "0 0 1 1 *",
		Feeds:      feedURLs,
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() {
		runErr <- run(ctx, []string{"-config", configPath})
	}()

	waitForFile(t, outputPath)
	cancel()

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("run did not stop after context cancellation")
	}

	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	doc, err := goquery.NewDocumentFromReader(file)
	if err != nil {
		t.Fatal(err)
	}

	if title := strings.TrimSpace(doc.Find("title").Text()); title != "Simple RSS" {
		t.Fatalf("title = %q, want Simple RSS", title)
	}
	if h1 := strings.TrimSpace(doc.Find("h1").First().Text()); h1 != "Feeds" {
		t.Fatalf("h1 = %q, want Feeds", h1)
	}
	if sections := doc.Find("section").Length(); sections == 0 {
		t.Fatal("expected at least one feed section")
	}
	if links := doc.Find("section li a[href]").Length(); links == 0 {
		t.Fatal("expected at least one article link")
	}
	if metas := doc.Find("section li .meta").Length(); metas == 0 {
		t.Fatal("expected at least one article metadata element")
	}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()

	deadline := time.After(1 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %s", path)
		case <-ticker.C:
			info, err := os.Stat(path)
			if err == nil && info.Size() > 0 {
				return
			}
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(fmt.Errorf("stat %s: %w", path, err))
			}
		}
	}
}
