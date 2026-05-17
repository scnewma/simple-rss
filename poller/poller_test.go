package poller

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"simple-rss/storage"
)

func TestPollStoresFeedArticles(t *testing.T) {
	cases := []feedFixture{
		{
			name:        "matklad atom feed",
			path:        "testdata/matklad-feed.xml",
			contentType: "application/atom+xml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			feed := serveFeedFixture(t, tc)
			store := openTestStore(t)
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
			poller := New([]string{feed.URL}, store, logger)

			summary := poller.Poll(context.Background())
			assertSuccessfulPoll(t, summary, 1)

			articles, err := store.ListRecentArticles(context.Background(), time.Time{})
			if err != nil {
				t.Fatal(err)
			}
			if len(articles) != summary.ItemsInserted {
				t.Fatalf("stored articles = %d, want %d", len(articles), summary.ItemsInserted)
			}

			assertArticlesLookStored(t, articles, feed.URL)
		})
	}
}

type feedFixture struct {
	name        string
	path        string
	contentType string
}

type servedFeed struct {
	URL string
}

func serveFeedFixture(t *testing.T, fixture feedFixture) servedFeed {
	t.Helper()

	feedXML, err := os.ReadFile(fixture.path)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", fixture.contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(feedXML)
	}))
	t.Cleanup(server.Close)

	return servedFeed{URL: server.URL}
}

func openTestStore(t *testing.T) *storage.Store {
	t.Helper()

	store, err := storage.Open(t.TempDir() + "/simple-rss.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	})

	return store
}

func assertSuccessfulPoll(t *testing.T, summary Summary, wantFeedsAttempted int) {
	t.Helper()

	if summary.FeedsAttempted != wantFeedsAttempted {
		t.Fatalf("FeedsAttempted = %d, want %d", summary.FeedsAttempted, wantFeedsAttempted)
	}
	if len(summary.FailedFeeds) != 0 {
		t.Fatalf("FailedFeeds = %v, want none", summary.FailedFeeds)
	}
	if summary.ItemsInserted == 0 {
		t.Fatal("ItemsInserted = 0, want at least one article")
	}
}

func assertArticlesLookStored(t *testing.T, articles []storage.Article, wantFeedURL string) {
	t.Helper()

	if len(articles) == 0 {
		t.Fatal("no stored articles")
	}

	for _, article := range articles {
		if article.Title == "" {
			t.Fatal("article has empty title")
		}
		if article.Link == "" {
			t.Fatal("article has empty link")
		}
		if article.SourceFeed.URL != wantFeedURL {
			t.Fatalf("SourceFeed.URL = %q, want %q", article.SourceFeed.URL, wantFeedURL)
		}
		if article.SourceFeed.Title == "" {
			t.Fatal("SourceFeed.Title is empty")
		}
	}
}
