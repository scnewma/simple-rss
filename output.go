package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
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
              <div class="meta">{{.Feed.Title}} · {{formatDate .PublishedAt}}</div>
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

func WriteJSON(w io.Writer, feeds []*Feed, groups []GroupConfig) error {
	data := struct {
		Groups []group `json:"groups"`
	}{
		Groups: groupArticles(feeds, groups),
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func WriteHTML(w io.Writer, feeds []*Feed, groups []GroupConfig) error {
	t, err := template.New("index").Funcs(template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Local().Format("Jan 2, 2006")
		},
	}).Parse(pageHTML)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	articleGroups := groupArticles(feeds, groups)

	pageData := struct{ Groups []group }{
		Groups: articleGroups,
	}

	buf := bytes.NewBuffer(nil)
	if err := t.Execute(buf, pageData); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	_, err = w.Write(buf.Bytes())
	return err
}

type group struct {
	Title    string          `json:"title"`
	Articles []outputArticle `json:"articles"`

	shouldAdd func(publishedAt time.Time) bool `json:"-"`
}

type outputArticle struct {
	Link        string     `json:"link"`
	Title       string     `json:"title"`
	PublishedAt time.Time  `json:"publishedAt"`
	Feed        outputFeed `json:"feed"`
}

type outputFeed struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func groupArticles(feeds []*Feed, configs []GroupConfig) []group {
	now := clock.Now()

	groups := make([]group, 0, len(configs))
	for _, cfg := range configs {
		maxAge := cfg.MaxAge.Duration()
		groups = append(groups, group{
			Title: cfg.Title,
			shouldAdd: func(publishedAt time.Time) bool {
				return maxAge == 0 || now.Sub(publishedAt) <= maxAge
			},
		})
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
					Feed: outputFeed{
						Title: feed.Title,
						URL:   feed.URL,
					},
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
