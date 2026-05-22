package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type staticClock struct {
	now time.Time
}

func (c staticClock) Now() time.Time {
	return c.now
}

func TestApp(t *testing.T) {
	originalClock := clock
	clock = staticClock{now: time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)}
	t.Cleanup(func() { clock = originalClock })

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
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := Config{
		Feeds: feedURLs,
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, err := os.CreateTemp(tmpDir, "stdout-*")
	if err != nil {
		t.Fatal(err)
	}
	defer stdout.Close()

	originalStdout := os.Stdout
	os.Stdout = stdout
	t.Cleanup(func() { os.Stdout = originalStdout })

	if err := run(context.Background(), []string{"-config", configPath}); err != nil {
		t.Fatal(err)
	}

	if _, err := stdout.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	html, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
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

	doc.Find("section").Each(func(i int, section *goquery.Selection) {
		if articles := section.Find("li").Length(); articles == 0 {
			group := strings.TrimSpace(section.Find("h2").First().Text())
			t.Fatalf("expected group %d (%q) to be non-empty", i, group)
		}
	})
}
