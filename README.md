# Simple RSS

A simple RSS server that just displays a page of links. Configure feeds in JSON and they will update daily.

## Configuration

Reads `config.json` by default. Use `-config path/to/config.json` to choose a different file. If the file does not exist, built-in defaults are used.

```json
{
  "pollCron": "0 0 0 * * *",
  "displayDays": 90,
  "listenAddr": ":8080",
  "databasePath": "simple-rss.db",
  "feeds": []
}
```

Options:

- `pollCron`: six-field cron schedule, including seconds. Defaults to daily at midnight.
- `displayDays`: number of days of articles to show. Defaults to `90`.
- `listenAddr`: HTTP listen address. Defaults to `:8080`.
- `databasePath`: SQLite database path. Defaults to `simple-rss.db`.
- `feeds`: RSS/Atom feed URLs. Must be HTTP or HTTPS.

## Inspiration

https://matklad.github.io/2025/06/26/rssssr.html
