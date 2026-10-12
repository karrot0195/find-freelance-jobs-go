# Part-Time Job Aggregator

A desktop job aggregator built in Go for tracking freelance opportunities from supported providers. The current implementation focuses on **Freelancer Việt** and stores the results in a local SQLite database, with a Gio-based interface for browsing and refreshing jobs.

## What the app does

- Lets the user add a provider account/email from the UI
- Stores enabled accounts in SQLite
- Runs a "Start All" sweep to fetch jobs from every enabled provider
- Parses provider-specific HTML into a consistent `JobItem` shape
- Saves jobs with a unique URL constraint so duplicates are ignored
- Displays the latest job results in a table view

## Tech stack

- **Language:** Go
- **UI:** [Gio](https://gioui.org/)
- **Database:** SQLite via `modernc.org/sqlite`
- **Scraping:** Standard library `net/http` + Go HTML parsing (`golang.org/x/net/html`)
- **Platforms:** Linux, macOS, Windows

## Project structure

```text
find-freelance-jobs/
├── main.go                     # App bootstrap and database initialization
├── makefile                    # Build target: `make build`
├── go.mod
├── go.sum
├── jobs.db                     # Local SQLite database generated at runtime
├── bin/
│   └── job-aggregator         # Built binary
├── internal/
│   ├── db/
│   │   └── database.go         # Account/job persistence helpers
│   ├── scraper/
│   │   ├── provider.go         # Provider registry contract
│   │   └── freelancerviet.go   # Freelancer Việt scraper implementation
│   └── ui/
│       └── app.go              # Gio dashboard and account form UI
├── docs/
│   ├── FLOW.md                 # End-to-end runtime workflow
│   └── providers/
│       ├── freelancer-viet.spec.md
│       └── template.md
├── README.md
└── .gitignore
```

## Core architecture

### Provider registry

`internal/scraper/provider.go` defines the `JobProvider` interface:

```go
type JobProvider interface {
    Name() string
    FetchJobs(username string) ([]JobItem, error)
}
```

Each provider registers itself in an `init()` function and is exposed via global registry functions (`Register`, `FetchJobs`, `GetSupportedProviders`). This keeps the UI and database layer independent from site-specific scraping logic.

### Scraping flow

1. The user opens the dashboard and clicks **Start All**.
2. `db.ClearJobs()` removes stale job rows.
3. `db.GetAccounts()` loads all enabled accounts.
4. For each enabled account, the app calls `scraper.FetchJobs(providerName, username)`.
5. The selected provider fetches and parses HTML into a standardized `JobItem` list.
6. Each item is saved with `db.SaveJob(...)` using an `INSERT OR IGNORE` pattern for deduplication by URL.

### UI flow

`internal/ui/app.go` manages two screens:

- `ScreenDashboard`: shows the start button, account summary, and jobs table
- `ScreenAddAccount`: captures a username/email and stores it in the SQLite accounts table

## Supported providers

At the moment, the app includes support for:

- `Freelancer Việt`

Provider-specific parsing notes and DOM selectors live under `docs/providers/`.

## Development and run instructions

### Build

```bash
make build
```

### Run

```bash
./bin/job-aggregator
```

## Database schema

The app creates the following tables when it starts:

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
    provider TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Adding a new provider

1. Create a provider spec in `docs/providers/` following `template.md`
2. Add a new scraper implementation under `internal/scraper/`
3. Implement `Name()` and `FetchJobs(username string)`
4. Register it in `init()` with `Register(...)`
5. Rebuild and use the app to fetch jobs from the new provider

## Notes

- The scraper currently targets `https://freelancerviet.vn/viec-lam-freelance.html`
- If the site markup changes, it falls back to a sample mock job record so the app remains functional in restricted environments
- Results are stored locally in `./jobs.db`, which is created automatically on first run