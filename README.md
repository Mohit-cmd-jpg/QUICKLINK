# QuickLink

> A fast, focused URL shortener built with Go.

QuickLink turns long web addresses into compact links that are easy to share,
remember, and track. It provides a responsive browser interface, a JSON API,
HTTP redirects, basic visit statistics, and SQLite-backed persistence.

## Live deployment

**Production URL:** [https://quicklink-n9t2.onrender.com](https://quicklink-n9t2.onrender.com)

The service is deployed on Render from the `main` branch of
[Mohit-cmd-jpg/QUICKLINK](https://github.com/Mohit-cmd-jpg/QUICKLINK).
Render supplies the port through the `PORT` environment variable and uses the
free-tier filesystem. Because the free tier has ephemeral storage, SQLite data
may be reset after a service restart or redeploy.

## Features

- Create unique short links from valid HTTP or HTTPS URLs.
- Redirect short codes to their original destinations.
- Count visits for every resolved short link.
- View creation time and visit count through the stats endpoint.
- Validate malformed, empty, and unsupported URL inputs.
- Responsive interface for desktop and mobile screens.
- Copy generated links with one click.
- Keyboard shortcut: `Ctrl/Cmd + K` focuses the URL field.
- Health endpoint for deployment monitoring.
- Automated Go tests for API and validation behavior.

## Technology

- **Language:** Go 1.22+
- **HTTP:** Go standard library `net/http`
- **Database:** SQLite through `modernc.org/sqlite`
- **Frontend:** HTML, CSS, and vanilla JavaScript
- **Hosting:** Render Blueprint

## Run locally

Prerequisites:

- Go 1.22 or newer

From the project directory:

```powershell
go mod tidy
go test ./...
go run .
```

Open [http://localhost:8080](http://localhost:8080).

The local database is created as `quicklink.db`. It is ignored by Git.

### Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | Port supplied by Render or another host |
| `QUICKLINK_ADDR` | `:<PORT>` | Full listen address; overrides `PORT` |
| `QUICKLINK_DATA_DIR` | `.` | Directory containing `quicklink.db` |

Example:

```powershell
$env:PORT = "8080"
$env:QUICKLINK_DATA_DIR = ".\data"
go run .
```

## API

### Create a short link

```http
POST /api/urls
Content-Type: application/json

{"url":"https://example.com/articles/a-long-url"}
```

Successful response (`201 Created`):

```json
{
  "code": "Ab3xY7k",
  "short_url": "http://localhost:8080/Ab3xY7k"
}
```

Invalid or unsupported URLs return `400 Bad Request` with an `error` field.

### Resolve a short link

```http
GET /{code}
```

Known codes return `302 Found`, increment the visit count, and redirect to the
original URL. Unknown codes return `404 Not Found`.

### Read link statistics

```http
GET /api/urls/{code}/stats
```

Successful response (`200 OK`):

```json
{
  "short_code": "Ab3xY7k",
  "original_url": "https://example.com/articles/a-long-url",
  "created_at": "2026-11-15T12:00:00Z",
  "visit_count": 3
}
```

### Health check

```http
GET /health
```

Returns `200 OK`:

```json
{"status":"ok"}
```

## Project structure

```text
.
├── main.go          # HTTP server, routes, store, validation, and handlers
├── main_test.go     # API and validation tests
├── go.mod           # Go module and dependencies
├── render.yaml      # Render Blueprint deployment configuration
└── web/
    ├── index.html   # Browser interface and client-side API logic
    └── style.css    # Responsive visual design
```

## Architecture

1. The browser sends a URL to `POST /api/urls`.
2. The Go HTTP layer validates the request and delegates persistence to the
   SQLite store.
3. The store generates a collision-resistant short code and saves the mapping.
4. The browser receives a shareable short URL.
5. Requests to `/{code}` look up the mapping, record a visit, and issue a
   redirect.
6. `/api/urls/{code}/stats` exposes basic link metadata and visit count.

## Deploy on Render

The repository includes [`render.yaml`](./render.yaml). To deploy or redeploy:

1. Open the [Render Dashboard](https://dashboard.render.com/).
2. Select **New → Blueprint**.
3. Connect the `Mohit-cmd-jpg/QUICKLINK` repository.
4. Select the `main` branch.
5. Keep the Blueprint path as `render.yaml`.
6. Set the Blueprint name to `quicklink`.
7. Select **Deploy Blueprint**.

The Blueprint uses:

- Build command: `go build -o quicklink .`
- Start command: `./quicklink`
- Health check: `/health`
- Free web-service plan
- Ephemeral SQLite storage at `/tmp/quicklink`

## Scope and limitations

The current version intentionally does not include user accounts, custom
aliases, branded domains, advanced analytics, geographic tracking, billing,
QR-code generation, or distributed multi-instance storage.

## Team

**Institution:** K.R. Mangalam University, School of Engineering & Technology

**Course:** ENSP465 — New Age Programming Languages Lab

**Project:** QuickLink: URL Shortener Using Go

**Semester:** VII

**Academic year:** 2026–2027

**Section:** 1

**Faculty:** Dr. Hemraj Kumawat

| Member | Enrollment number | Responsibility |
| --- | --- | --- |
| **Mohit** (Team Leader) | 2301010310 | Coordinate milestones, define API contracts, integrate modules, and lead the final demonstration |
| **Prabhat Nagar** | 2301010316 | Design and implement the database schema and persistence operations; test mapping creation and lookup |
| **Vedaan Sharma** | 2301010309 | Build the user interface, connect it to the create-link endpoint, and verify user-facing messages |
| **Pranav Panday** | 2301010094 | Implement redirect and validation paths, contribute error handling, and test unknown or invalid codes |

All members review design decisions, contribute test cases, document individual
work, and prepare for individual viva questions.

## License

This project was created as an academic capstone for ENSP465.
