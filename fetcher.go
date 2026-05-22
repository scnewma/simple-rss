package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

const feedTimeout = 20 * time.Second

type Feed struct {
	URL      string
	Title    string
	Articles []Article
}

type Article struct {
	Title       string
	Link        string
	PublishedAt time.Time
}

type FeedError struct {
	FeedURL string
	Err     error
}

func (e FeedError) Error() string {
	return fmt.Sprintf("fetch %q: %s", e.FeedURL, e.Err)
}

type Fetcher struct {
	parser *gofeed.Parser

	now func() time.Time
}

func (f *Fetcher) FetchAll(ctx context.Context, feedURLs []string) ([]*Feed, error) {
	f.init()

	telemetry := instrument(LoggerFromContext(ctx))
	defer telemetry.Close()

	var feeds []*Feed
	var merr error

	for _, feedURL := range feedURLs {
		select {
		case <-ctx.Done():
			return feeds, errors.Join(merr, ctx.Err())
		default:
		}

		span := telemetry.Start(feedURL)

		feed, err := f.FetchOne(ctx, feedURL)
		if err != nil {
			merr = errors.Join(merr, FeedError{feedURL, err})
			span.Error()
			continue
		}
		feeds = append(feeds, feed)

		span.Finish(feed)
	}

	return feeds, merr
}

func (f *Fetcher) FetchOne(ctx context.Context, feedURL string) (*Feed, error) {
	f.init()

	feedCtx, cancel := context.WithTimeout(ctx, feedTimeout)
	defer cancel()

	fetched, err := f.parser.ParseURLWithContext(feedURL, feedCtx)
	if err != nil {
		return nil, fmt.Errorf("fetching feed %q failed: %w", feedURL, err)
	}

	feed := &Feed{
		URL:   feedURL,
		Title: strings.TrimSpace(fetched.Title),
	}
	for _, item := range fetched.Items {
		publishedAt := f.now()
		if item.PublishedParsed != nil {
			publishedAt = *item.PublishedParsed
		}

		feed.Articles = append(feed.Articles, Article{
			Title:       item.Title,
			Link:        item.Link,
			PublishedAt: publishedAt,
		})
	}
	return feed, nil
}

func (f *Fetcher) init() {
	if f.now == nil {
		f.now = time.Now
	}
	if f.parser == nil {
		f.parser = gofeed.NewParser()
	}
}

type fetcherTelemetry struct {
	logger   *slog.Logger
	begin    time.Time
	nfailed  int
	nsuccess int
}

func instrument(logger *slog.Logger) *fetcherTelemetry {
	return &fetcherTelemetry{logger: logger, begin: time.Now()}
}

func (t *fetcherTelemetry) Start(feedURL string) *span {
	return &span{
		telemetry: t,
		logger:    t.logger,
		feedURL:   feedURL,
		begin:     time.Now(),
	}
}

func (t *fetcherTelemetry) Close() {
	elapsed := time.Now().Sub(t.begin)
	t.logger.Info("fetched feeds", "duration", elapsed, "feeds_attempted", t.nsuccess+t.nfailed, "feeds_failed", t.nfailed)
}

type span struct {
	telemetry *fetcherTelemetry
	logger    *slog.Logger
	feedURL   string
	begin     time.Time
}

func (s *span) Error() {
	s.telemetry.nfailed += 1

	s.logger.Error("fetching feed failed", "feed", s.feedURL, "duration", s.elapsed())
}

func (s *span) Finish(feed *Feed) {
	s.telemetry.nsuccess += 1

	s.logger.Info("fetched feed", "feed", s.feedURL, "duration", s.elapsed(), "title", feed.Title, "articles", len(feed.Articles))
}

func (s *span) elapsed() time.Duration {
	return time.Now().Sub(s.begin)
}
