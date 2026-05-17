package poller

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"simple-rss/storage"
)

type articleStore interface {
	InsertArticle(ctx context.Context, feed storage.Feed, article storage.Article) (bool, error)
}

type Poller struct {
	feeds  []string
	store  articleStore
	parser *gofeed.Parser
	logger *slog.Logger
}

type Summary struct {
	FeedsAttempted    int
	FailedFeeds       []string
	ItemsInserted     int
	DuplicatesSkipped int
	ItemsSkipped      int
}

type feedSummary struct {
	ItemsSeen         int
	ItemsInserted     int
	DuplicatesSkipped int
	ItemsSkipped      int
	Duration          time.Duration
}

func New(feeds []string, store articleStore, logger *slog.Logger) *Poller {
	return &Poller{
		feeds:  feeds,
		store:  store,
		parser: gofeed.NewParser(),
		logger: logger,
	}
}

func (p *Poller) Poll(ctx context.Context) Summary {
	startedAt := time.Now()
	fetchedAt := startedAt
	var summary Summary

	for _, feedURL := range p.feeds {
		summary.FeedsAttempted++
		feedStartedAt := time.Now()
		var feedSummary feedSummary

		feedCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		feed, err := p.parser.ParseURLWithContext(feedURL, feedCtx)
		cancel()
		if err != nil {
			feedSummary.Duration = time.Since(feedStartedAt)
			summary.FailedFeeds = append(summary.FailedFeeds, feedURL)
			p.logger.Error("poll feed failed", "feed", feedURL, "duration", feedSummary.Duration, "error", err)
			continue
		}

		storedFeed := storage.Feed{URL: feedURL, Title: strings.TrimSpace(feed.Title)}
		feedSummary.ItemsSeen = len(feed.Items)
		for _, item := range feed.Items {
			article, ok := normalizeItem(item, fetchedAt)
			if !ok {
				feedSummary.ItemsSkipped++
				continue
			}

			inserted, err := p.store.InsertArticle(ctx, storedFeed, article)
			if err != nil {
				p.logger.Error("insert article", "feed", feedURL, "link", article.Link, "error", err)
				continue
			}
			if inserted {
				feedSummary.ItemsInserted++
			} else {
				feedSummary.DuplicatesSkipped++
			}
		}

		feedSummary.Duration = time.Since(feedStartedAt)
		summary.ItemsInserted += feedSummary.ItemsInserted
		summary.DuplicatesSkipped += feedSummary.DuplicatesSkipped
		summary.ItemsSkipped += feedSummary.ItemsSkipped
		p.logger.Info("poll feed complete",
			"feed", feedURL,
			"feed_title", feed.Title,
			"duration", feedSummary.Duration,
			"items_seen", feedSummary.ItemsSeen,
			"items_inserted", feedSummary.ItemsInserted,
			"duplicates_skipped", feedSummary.DuplicatesSkipped,
			"items_skipped", feedSummary.ItemsSkipped,
		)
	}

	p.logger.Info("poll complete",
		"duration", time.Since(startedAt),
		"feeds_attempted", summary.FeedsAttempted,
		"feeds_failed", len(summary.FailedFeeds),
		"items_inserted", summary.ItemsInserted,
		"duplicates_skipped", summary.DuplicatesSkipped,
		"items_skipped", summary.ItemsSkipped,
	)

	return summary
}

func normalizeItem(item *gofeed.Item, fetchedAt time.Time) (storage.Article, bool) {
	link := strings.TrimSpace(item.Link)
	dedupeKey := canonicalizeURL(link)
	if dedupeKey == "" {
		dedupeKey = strings.TrimSpace(item.GUID)
	}
	if dedupeKey == "" {
		return storage.Article{}, false
	}

	publishedAt := fetchedAt
	if item.PublishedParsed != nil {
		publishedAt = *item.PublishedParsed
	}

	title := strings.TrimSpace(item.Title)
	if title == "" {
		title = link
	}

	return storage.Article{
		Title:       title,
		Link:        link,
		DedupeKey:   dedupeKey,
		PublishedAt: publishedAt,
		FetchedAt:   fetchedAt,
	}, true
}

func canonicalizeURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}

	u.Fragment = ""
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	return u.String()
}

func (s Summary) String() string {
	return fmt.Sprintf("attempted=%d failed=%d inserted=%d duplicates=%d skipped=%d", s.FeedsAttempted, len(s.FailedFeeds), s.ItemsInserted, s.DuplicatesSkipped, s.ItemsSkipped)
}
