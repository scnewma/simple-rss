package main

import "time"

func filterArticlesByMaxAge(feeds []*Feed, maxAge time.Duration) {
	now := clock.Now()
	for _, feed := range feeds {
		articles := feed.Articles[:0]
		for _, article := range feed.Articles {
			if now.Sub(article.PublishedAt) <= maxAge {
				articles = append(articles, article)
			}
		}
		feed.Articles = articles
	}
}
