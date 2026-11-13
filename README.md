# QuickLink

QuickLink is a small Go URL shortener backed by SQLite.

## Run

```powershell
go mod tidy
go run .
```

Open [http://localhost:8080](http://localhost:8080).

The database is stored in `quicklink.db`. Set `QUICKLINK_ADDR` to change the
listen address or `QUICKLINK_DATA_DIR` to change the database directory.

## API

- `POST /api/urls` with `{"url":"https://example.com"}` creates a link.
- `GET /{code}` redirects to the original URL and records a visit.
- `GET /api/urls/{code}/stats` returns link metadata and visit count.
- `GET /health` checks service availability.
