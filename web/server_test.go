package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"simple-rss/storage"
)

func TestHandlerRendersGroupedArticleLinks(t *testing.T) {
	store := fakeArticleStore{
		articles: []storage.Article{
			article("Today Article", "https://example.com/today", daysAgo(0)),
			article("Week Article", "https://example.com/week", daysAgo(3)),
			article("Month Article", "https://example.com/month", daysAgo(10)),
			article("Older Article", "https://example.com/older", daysAgo(40)),
		},
	}
	response := requestHome(t, store)

	assertContains(t, response, "Today")
	assertContains(t, response, "Last 7 Days")
	assertContains(t, response, "Last 30 Days")
	assertContains(t, response, "Older")
	assertContains(t, response, "Today Article")
	assertContains(t, response, "Week Article")
	assertContains(t, response, "Month Article")
	assertContains(t, response, "Older Article")
	assertContains(t, response, `target="_blank"`)
	assertContains(t, response, `rel="noopener noreferrer"`)
	assertContains(t, response, "Example Feed")
}

func TestHandlerRendersEmptyState(t *testing.T) {
	response := requestHome(t, fakeArticleStore{})

	assertContains(t, response, "No articles yet")
}

type fakeArticleStore struct {
	articles []storage.Article
}

func (s fakeArticleStore) ListRecentArticles(ctx context.Context, since time.Time) ([]storage.Article, error) {
	return s.articles, nil
}

func requestHome(t *testing.T, store fakeArticleStore) string {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	server := httptest.NewServer(Handler(store, 90, logger))
	t.Cleanup(server.Close)

	response, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.StatusCode, body)
	}

	return string(body)
}

func article(title string, link string, publishedAt time.Time) storage.Article {
	return storage.Article{
		Title: title,
		Link:  link,
		SourceFeed: storage.Feed{
			URL:   "https://example.com/feed.xml",
			Title: "Example Feed",
		},
		PublishedAt: publishedAt,
	}
}

func daysAgo(days int) time.Time {
	return time.Now().AddDate(0, 0, -days)
}

func assertContains(t *testing.T, haystack string, needle string) {
	t.Helper()

	if !strings.Contains(haystack, needle) {
		t.Fatalf("response does not contain %q\n\n%s", needle, haystack)
	}
}
