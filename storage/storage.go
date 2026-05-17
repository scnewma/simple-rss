package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

type Feed struct {
	URL   string
	Title string
}

type Article struct {
	Title       string
	Link        string
	DedupeKey   string
	SourceFeed  Feed
	PublishedAt time.Time
	FetchedAt   time.Time
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	store := &Store{db: db}
	if err := store.createSchema(context.Background()); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) InsertArticle(ctx context.Context, feed Feed, article Article) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin insert article: %w", err)
	}
	defer tx.Rollback()

	feedID, err := upsertFeed(ctx, tx, feed)
	if err != nil {
		return false, err
	}

	result, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO articles (
			feed_id, title, link, dedupe_key, published_at, fetched_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`,
		feedID,
		article.Title,
		article.Link,
		article.DedupeKey,
		article.PublishedAt.UTC().Format(time.RFC3339Nano),
		article.FetchedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return false, fmt.Errorf("insert article: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("insert article rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit insert article: %w", err)
	}

	return rowsAffected > 0, nil
}

func (s *Store) ListRecentArticles(ctx context.Context, since time.Time) ([]Article, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT articles.title, articles.link, articles.dedupe_key, feeds.url, feeds.title, articles.published_at, articles.fetched_at
		FROM articles
		JOIN feeds ON feeds.id = articles.feed_id
		WHERE articles.published_at >= ?
		ORDER BY articles.published_at DESC
	`, since.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("list recent articles: %w", err)
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		var article Article
		var publishedAt string
		var fetchedAt string
		if err := rows.Scan(
			&article.Title,
			&article.Link,
			&article.DedupeKey,
			&article.SourceFeed.URL,
			&article.SourceFeed.Title,
			&publishedAt,
			&fetchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}

		article.PublishedAt, err = time.Parse(time.RFC3339Nano, publishedAt)
		if err != nil {
			return nil, fmt.Errorf("parse published time: %w", err)
		}
		article.FetchedAt, err = time.Parse(time.RFC3339Nano, fetchedAt)
		if err != nil {
			return nil, fmt.Errorf("parse fetched time: %w", err)
		}

		articles = append(articles, article)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate articles: %w", err)
	}

	return articles, nil
}

func upsertFeed(ctx context.Context, tx *sql.Tx, feed Feed) (int64, error) {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO feeds (url, title)
		VALUES (?, ?)
		ON CONFLICT(url) DO UPDATE SET title = excluded.title
	`, feed.URL, feed.Title)
	if err != nil {
		return 0, fmt.Errorf("upsert feed: %w", err)
	}

	var feedID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM feeds WHERE url = ?`, feed.URL).Scan(&feedID); err != nil {
		return 0, fmt.Errorf("select feed: %w", err)
	}

	return feedID, nil
}

func (s *Store) createSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS feeds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS articles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feed_id INTEGER NOT NULL REFERENCES feeds(id),
			title TEXT NOT NULL,
			link TEXT NOT NULL,
			dedupe_key TEXT NOT NULL UNIQUE,
			published_at TEXT NOT NULL,
			fetched_at TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS articles_published_at_idx ON articles (published_at DESC);
		CREATE INDEX IF NOT EXISTS articles_feed_id_idx ON articles (feed_id);
	`)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	return nil
}
