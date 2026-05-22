package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		Feeds: feedURLs,
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run(context.Background(), []string{"-config", configPath, "-output", outputPath}); err != nil {
		t.Fatal(err)
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
