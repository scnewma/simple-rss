package web

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"simple-rss/storage"
)

type articleStore interface {
	ListRecentArticles(ctx context.Context, since time.Time) ([]storage.Article, error)
}

type pageData struct {
	Groups []articleGroup
}

type articleGroup struct {
	Title    string
	Articles []storage.Article
}

func Handler(store articleStore, displayDays int, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex(store, displayDays, logger))
	return mux
}

func handleIndex(store articleStore, displayDays int, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		since := now.AddDate(0, 0, -displayDays)
		articles, err := store.ListRecentArticles(r.Context(), since)
		if err != nil {
			logger.Error("list articles", "error", err)
			http.Error(w, "failed to load articles", http.StatusInternalServerError)
			return
		}

		t := template.Must(template.New("index").Funcs(template.FuncMap{
			"formatDate": formatDate,
		}).Parse(pageHTML))
		if err := t.Execute(w, pageData{Groups: groupArticles(articles, now)}); err != nil {
			logger.Error("render index", "error", err)
		}
	}
}

func groupArticles(articles []storage.Article, now time.Time) []articleGroup {
	groups := []articleGroup{
		{Title: "Today"},
		{Title: "Last 7 Days"},
		{Title: "Last 30 Days"},
		{Title: "Older"},
	}

	for _, article := range articles {
		index := articleGroupIndex(article.PublishedAt, now)
		groups[index].Articles = append(groups[index].Articles, article)
	}

	var nonEmptyGroups []articleGroup
	for _, group := range groups {
		if len(group.Articles) > 0 {
			nonEmptyGroups = append(nonEmptyGroups, group)
		}
	}

	return nonEmptyGroups
}

func articleGroupIndex(publishedAt time.Time, now time.Time) int {
	publishedLocal := publishedAt.Local()
	nowLocal := now.Local()
	if publishedLocal.Year() == nowLocal.Year() && publishedLocal.YearDay() == nowLocal.YearDay() {
		return 0
	}

	age := now.Sub(publishedAt)
	if age < 7*24*time.Hour {
		return 1
	}
	if age < 30*24*time.Hour {
		return 2
	}
	return 3
}

func formatDate(t time.Time) string {
	return t.Local().Format("Jan 2, 2006")
}

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
              <div class="meta">{{.SourceFeed.Title}} · {{formatDate .PublishedAt}}</div>
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
