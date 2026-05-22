# Simple RSS

A simple RSS tool that fetches configured feeds and writes a static HTML page of links.

## Usage

```sh
simple-rss -config config.json > index.html
```

Flags:

- `-config`: path to config file. Defaults to `config.json`.
- `-format`: output format, `html` or `json`. Defaults to `html`.
- `-max-age`: maximum article age to include as a Go duration, like `24h` or `168h`. Defaults to no limit.

## Configuration

The config file only contains the feed list:

```json
{
  "feeds": [
    "https://example.com/feed.xml"
  ]
}
```

Feeds must be HTTP or HTTPS RSS/Atom feed URLs.

## Inspiration

https://matklad.github.io/2025/06/26/rssssr.html
