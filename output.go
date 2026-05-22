package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"time"
)

const pageHTML = `<!doctype html>
<html>
<head>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Simple RSS</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 760px; margin: 2rem auto; padding: 0 1rem; color: #1f2937; }
    h1 { margin-bottom: 2rem; }
    h2 { margin-top: 2rem; border-bottom: 1px solid #e5e7eb; padding-bottom: 0.35rem; }
    ul { list-style: none; padding: 0; }
    li { margin: 1rem 0; }
    a { color: #075985; font-weight: 650; text-decoration-thickness: 0.08em; }
    a:visited { color: #7353ba; }
    .meta { color: #6b7280; font-size: 0.9rem; margin-top: 0.25rem; }
  </style>
</head>
<body>
  <h1>Feeds</h1>
  {{if .Groups}}
    {{range .Groups}}
      <section>
        <h2>{{.Title}}</h2>
        <ul>
          {{range .Articles}}
            <li>
              <a href="{{.Link}}" target="_blank" rel="noopener noreferrer">{{.Title}}</a>
              <div class="meta">{{.FeedTitle}} · {{formatDate .PublishedAt}}</div>
            </li>
          {{end}}
        </ul>
      </section>
    {{end}}
  {{else}}
    <p>No articles yet. Feeds will appear here after the next scheduled poll.</p>
  {{end}}
</body>
</html>`

func WriteHTML(path string, feeds []*Feed) error {
	t, err := template.New("index").Funcs(template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Local().Format("Jan 2, 2006")
		},
	}).Parse(pageHTML)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	groups := groupArticles(feeds)

	pageData := struct{ Groups []group }{
		Groups: groups,
	}

	buf := bytes.NewBuffer(nil)
	if err := t.Execute(buf, pageData); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	return os.WriteFile(path, buf.Bytes(), 0o666)
}

type group struct {
	Title    string
	Articles []outputArticle

	shouldAdd func(publishedAt time.Time) bool
}

type outputArticle struct {
	Link        string
	Title       string
	PublishedAt time.Time
	FeedTitle   string
}

func groupArticles(feeds []*Feed) []group {
	now := clock.Now()

	// indexing assumes these are in increasing order
	groups := []group{
		{
			Title: "Today",
			shouldAdd: func(publishedAt time.Time) bool {
				p, n := publishedAt.Local(), now.Local()
				return p.Year() == n.Year() && p.YearDay() == n.YearDay()
			},
		},
		{
			Title: "Last 7 Days",
			shouldAdd: func(publishedAt time.Time) bool {
				return now.Sub(publishedAt) < 7*24*time.Hour
			},
		},
		{
			Title: "Last 30 Days",
			shouldAdd: func(publishedAt time.Time) bool {
				return now.Sub(publishedAt) < 30*24*time.Hour
			},
		},
		{
			Title: "Older",
			shouldAdd: func(publishedAt time.Time) bool {
				return true
			},
		},
	}

	hasArticles := false
	for _, feed := range feeds {
		for _, article := range feed.Articles {
			hasArticles = true
			for i := range groups {
				if !groups[i].shouldAdd(article.PublishedAt) {
					continue
				}

				groups[i].Articles = append(groups[i].Articles, outputArticle{
					Link:        article.Link,
					Title:       article.Title,
					PublishedAt: article.PublishedAt,
					FeedTitle:   feed.Title,
				})
				break
			}
		}
	}

	if !hasArticles {
		return nil
	}

	return groups
}
