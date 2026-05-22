package main

import (
	"bytes"
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
	configPath := testConfigPath(t)

	html := captureStdout(t, func() error {
		return run(context.Background(), []string{"-config", configPath})
	})

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
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

func TestAppJSON(t *testing.T) {
	configPath := testConfigPath(t)

	output := captureStdout(t, func() error {
		return run(context.Background(), []string{"-config", configPath, "-format", "json"})
	})

	feeds := decodeFeeds(t, output)
	if len(feeds) == 0 {
		t.Fatal("expected at least one feed")
	}
	if len(feeds[0].Articles) == 0 {
		t.Fatal("expected at least one article")
	}
}

func TestAppMaxAge(t *testing.T) {
	configPath := testConfigPath(t)
	maxAge := 7 * 24 * time.Hour

	output := captureStdout(t, func() error {
		return run(context.Background(), []string{"-config", configPath, "-format", "json", "-max-age", maxAge.String()})
	})

	feeds := decodeFeeds(t, output)
	if len(feeds) == 0 || len(feeds[0].Articles) == 0 {
		t.Fatal("expected filtered output to include articles")
	}
	for _, feed := range feeds {
		for _, article := range feed.Articles {
			if age := clock.Now().Sub(article.PublishedAt); age > maxAge {
				t.Fatalf("article %q age = %s, want <= %s", article.Title, age, maxAge)
			}
		}
	}
}

func decodeFeeds(t *testing.T, output []byte) []Feed {
	t.Helper()

	var feeds []Feed
	if err := json.Unmarshal(output, &feeds); err != nil {
		t.Fatal(err)
	}
	return feeds
}

func testConfigPath(t *testing.T) string {
	t.Helper()

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

	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := Config{Feeds: feedURLs}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	return configPath
}

func captureStdout(t *testing.T, fn func() error) []byte {
	t.Helper()

	stdout, err := os.CreateTemp(t.TempDir(), "stdout-*")
	if err != nil {
		t.Fatal(err)
	}
	defer stdout.Close()

	originalStdout := os.Stdout
	os.Stdout = stdout
	t.Cleanup(func() { os.Stdout = originalStdout })

	if err := fn(); err != nil {
		t.Fatal(err)
	}

	if _, err := stdout.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatal(err)
	}
	return output
}
