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

	page := decodeJSONOutput(t, output)
	if len(page.Groups) == 0 {
		t.Fatal("expected at least one group")
	}
	for _, group := range page.Groups {
		if len(group.Articles) == 0 {
			t.Fatalf("expected %q group to include articles", group.Title)
		}
		assertArticlesNewestFirst(t, group.Articles)
	}
}

func TestAppCustomGroups(t *testing.T) {
	configPath := testConfigPath(t, GroupConfig{Title: "This Week", MaxAge: Duration(7 * 24 * time.Hour)})

	output := captureStdout(t, func() error {
		return run(context.Background(), []string{"-config", configPath, "-format", "json"})
	})

	page := decodeJSONOutput(t, output)
	if len(page.Groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(page.Groups))
	}
	if page.Groups[0].Title != "This Week" {
		t.Fatalf("group title = %q, want This Week", page.Groups[0].Title)
	}
	if len(page.Groups[0].Articles) == 0 {
		t.Fatal("expected custom group to include articles")
	}
}

func TestAppMaxAge(t *testing.T) {
	configPath := testConfigPath(t)
	maxAge := 7 * 24 * time.Hour

	output := captureStdout(t, func() error {
		return run(context.Background(), []string{"-config", configPath, "-format", "json", "-max-age", maxAge.String()})
	})

	page := decodeJSONOutput(t, output)
	if len(page.Groups) == 0 || len(page.Groups[0].Articles) == 0 {
		t.Fatal("expected filtered output to include articles")
	}
	for _, group := range page.Groups {
		for _, article := range group.Articles {
			if age := clock.Now().Sub(article.PublishedAt); age > maxAge {
				t.Fatalf("article %q age = %s, want <= %s", article.Title, age, maxAge)
			}
		}
	}
}

type jsonOutput struct {
	Groups []group `json:"groups"`
}

func decodeJSONOutput(t *testing.T, output []byte) jsonOutput {
	t.Helper()

	var page jsonOutput
	if err := json.Unmarshal(output, &page); err != nil {
		t.Fatal(err)
	}
	return page
}

func assertArticlesNewestFirst(t *testing.T, articles []outputArticle) {
	t.Helper()

	for i := 1; i < len(articles); i++ {
		if articles[i].PublishedAt.After(articles[i-1].PublishedAt) {
			t.Fatalf("articles not newest first: %q published at %s after previous article %q published at %s", articles[i].Title, articles[i].PublishedAt, articles[i-1].Title, articles[i-1].PublishedAt)
		}
	}
}

func testConfigPath(t *testing.T, groups ...GroupConfig) string {
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
	cfg := Config{Feeds: feedURLs, Groups: groups}
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
