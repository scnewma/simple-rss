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

The config file contains the feed list and optional article groups:

```json
{
  "feeds": [
    "https://example.com/feed.xml"
  ],
  "groups": [
    {
      "title": "Today",
      "maxAge": "24h"
    },
    {
      "title": "This Week",
      "maxAge": "168h"
    }
  ]
}
```

Feeds must be HTTP or HTTPS RSS/Atom feed URLs. If `groups` is omitted, the defaults are Today (`24h`), Last 7 Days (`168h`), Last 30 Days (`720h`), and Older (no max age).

## Inspiration

https://matklad.github.io/2025/06/26/rssssr.html
