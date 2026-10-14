# Part-Time Job Aggregator

A Go-based job aggregation project that monitors freelance postings across supported providers, stores results in SQLite, and exposes them through a small REST API plus a Gio desktop UI.

## What the app does

- Lets the user add provider accounts from the UI
- Persists enabled accounts in SQLite
- Crawls all enabled provider accounts when the user clicks Start All
- Normalizes each provider result into the same `JobItem` shape
- Deduplicates by job URL before inserting into the database
- Displays the jobs in a desktop dashboard and lets the user configure the API endpoint

## Tech stack

- Language: Go
- API: Gin
- UI: Gio
- Database: SQLite via `modernc.org/sqlite`
- Scraping: standard library HTTP + HTML parsing via `golang.org/x/net/html`
- Platforms: Linux, macOS, Windows

## Project structure

```text
find-freelance-jobs/
├── cmd/
│   ├── app/
│   │   └── main.go             # Gio desktop client entry point
│   └── server/
│       └── main.go             # Gin API entry point
├── internal/
│   ├── db/
│   │   └── database.go         # SQLite functions for accounts and jobs
│   ├── rest/
│   │   └── app_rest.go         # HTTP client used by the UI
│   ├── scraper/
│   │   ├── provider.go         # Provider registry contract
│   │   └── freelancerviet.go   # Freelancer Việt scraper
│   └── ui/
│       └── app.go              # Dashboard, settings, and add-account screens
├── docs/
│   ├── FLOW.md                 # End-to-end runtime flow
│   └── providers/
│       ├── freelancer-viet.spec.md
│       └── template.md
├── bin/
│   ├── job-aggregator-server   # Built server binary
│   └── job-aggregator-ui       # Built UI binary
├── go.mod
├── go.sum
├── jobs.db                     # Runtime SQLite database (created automatically)
├── makefile                    # Build and run helpers
├── README.md
├── .gitignore
└── .claude/
```

## Architecture

### Server layer

`cmd/server/main.go` starts a Gin HTTP server on `:8080` and exposes endpoints for:

- `GET /ping`
- `GET /GetAccounts`
- `POST /AddAccount`
- `GET /GetJobs`
- `POST /StartCrawlJobs`

The server reads and writes data through `internal/db`, and for each enabled account it calls `scraper.FetchJobs(providerName, username)`.

### UI layer

`cmd/app/main.go` boots the Gio app. `internal/ui/app.go` renders a dashboard, settings screen, and add-account form. The UI does not scrape directly; it calls the REST API through `internal/rest/app_rest.go`.

### Provider layer

`internal/scraper/provider.go` defines the shared provider contract:

```go
type JobProvider interface {
    Name() string
    FetchJobs(username string) ([]JobItem, error)
}
```

Registered providers are resolved through the global registry and normalized into the same `JobItem` model before they are saved.

## Supported provider

The current implementation includes:

- Freelancer Việt

The provider-specific selectors and extraction rules live in `docs/providers/freelancer-viet.spec.md`.

## Database schema

On startup, the app creates a local SQLite database at `./jobs.db` with these tables:

```sql
CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider TEXT NOT NULL,
    username TEXT NOT NULL,
    enabled BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    url TEXT UNIQUE NOT NULL,
    category TEXT,
    price TEXT,
    description TEXT,
    provider TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Development and run instructions

### Install dependencies

```bash
go mod download
```

### Build

```bash
make build
```

This produces:

```bash
./bin/job-aggregator-server
./bin/job-aggregator-ui
```

### Run everything together

```bash
make run
```

This starts both the API server and the desktop UI.

### Run manually

```bash
./bin/job-aggregator-server
./bin/job-aggregator-ui
```

Make sure the server is running before the UI makes API requests.

## End-to-end workflow

1. User launches the UI.
2. The UI loads the API base URL from settings and tests connectivity with `/ping`.
3. User adds one or more accounts via the add-account form.
4. User clicks Start All.
5. The UI posts to `POST /StartCrawlJobs`.
6. The server clears old job rows and fetches enabled accounts.
7. Each provider scrapes listings and normalizes them into `JobItem` records.
8. The server inserts jobs using `INSERT OR IGNORE` on `jobs.url` to avoid duplicate entries.
9. The UI refreshes and displays the latest results in the dashboard.

## Adding a new provider

1. Create a provider spec in `docs/providers/` using the template.
2. Implement a new scraper in `internal/scraper/` that satisfies the `JobProvider` interface.
3. Register it in an `init()` function.
4. Rebuild the binaries and test the new provider through the Start All workflow.

## Notes

- The current scraper targets `https://freelancerviet.vn/viec-lam-freelance.html`.
- If the site structure changes, the provider falls back to a sample job record so the UI can remain usable during testing.
- All results are stored locally in `./jobs.db` when the server starts.
- The project is designed as a small internal automation tool rather than a distributed web service.