# Application workflow: Part-Time Job Aggregator

This document describes the runtime flow implemented by the current codebase, from startup to database writes and provider scraping.

## 1. Startup sequence

```text
main.go
  -> db.InitDB()
  -> ui.Run()
```

On app launch:

- `main.go` initializes the SQLite database with `db.InitDB()`
- `db.InitDB()` creates the `accounts` and `jobs` tables if they do not already exist
- The Gio app window starts through `ui.Run()`
- The dashboard begins in `ScreenDashboard` and renders the current list of jobs and accounts

## 2. System components

### Database layer

`internal/db/database.go` owns persistence:

- `Account` records: `ID`, `Provider`, `Username`, `Enabled`
- `Job` records: `ID`, `Title`, `URL`, `Category`, `Price`, `Provider`, `CreatedAt`
- `InitDB()` sets up the SQLite schema
- `GetAccounts()`, `AddAccount()`, `ClearJobs()`, `SaveJob()`, and `GetJobs()` manage the app data

The app creates a local data file named `./jobs.db` when it starts.

### UI layer

`internal/ui/app.go` contains the application state machine:

- `ScreenDashboard`
- `ScreenAddAccount`

The dashboard renders:

- a **Start All** button
- an **Add Account** button
- a summary count for accounts and jobs
- a table of provider/title/category/price results

The add-account screen validates a non-empty input and writes a record to the `accounts` table before returning to the dashboard.

### Scraper layer

`internal/scraper/provider.go` defines the common interface used by all providers:

```go
type JobProvider interface {
    Name() string
    FetchJobs(username string) ([]JobItem, error)
}
```

The registry pattern keeps provider-specific logic separate from the UI and DB layers. Registered providers are discovered via `Register(...)`, `FetchJobs(...)`, and `GetSupportedProviders()`.

## 3. "Start All" workflow

When the user clicks **Start All** in the dashboard, the following sequence runs in `internal/ui/app.go`:

1. `db.ClearJobs()` removes old job rows from the database
2. `db.GetAccounts()` loads all saved accounts
3. The loop filters to accounts where `Enabled == true`
4. For each enabled account, the UI calls `scraper.FetchJobs(acc.Provider, acc.Username)`
5. The selected provider parses site HTML and returns `[]JobItem`
6. Each job is saved through `db.SaveJob(...)`
7. The UI reflects the new data in the table when the app redraws the frame

## 4. Current provider implementation

`internal/scraper/freelancerviet.go` defines `FreelancerVietProvider` and registers it with `init()`.

The provider performs:

- an HTTP `GET` to `https://freelancerviet.vn/viec-lam-freelance.html`
- HTML parsing using `golang.org/x/net/html`
- traversal for `.card-body` elements
- extraction of:
  - title + URL via `h3 a`
  - category via the first `li` inside the card
  - price via `p.font-size-24`

If the page structure does not match, it falls back to a sample job record to keep the UI operational in constrained test environments.

## 5. Data persistence rules

The `jobs` table uses a unique constraint on `url`:

```sql
INSERT OR IGNORE INTO jobs (title, url, category, price, provider) VALUES (?, ?, ?, ?, ?)
```

This prevents duplicate entries when the same job advertisement appears across refreshes.

## 6. Extending the app with a new provider

To add another provider, follow the same pattern used by `FreelancerVietProvider`:

1. Create a provider spec in `docs/providers/` using `template.md`
2. Implement a new type that satisfies the `JobProvider` interface
3. Register it in an `init()` function
4. Rebuild the app and use the **Start All** action to fetch jobs from it

No database schema change is required for a new provider, because all provider data is normalized to the same `JobItem` model.

## 7. End-to-end flow summary

```text
User launches app
  -> db.InitDB() creates local tables
  -> UI renders dashboard
  -> User adds account(s)
  -> User clicks Start All
  -> App clears old jobs
  -> Registry selects enabled provider(s)
  -> Scraper fetches and parses HTML
  -> Jobs saved to SQLite with unique URLs
  -> Dashboard shows updated table of results
```