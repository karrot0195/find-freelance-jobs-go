# Application workflow: Part-Time Job Aggregator

This document describes the current runtime flow for the project: the UI calls a REST API, the API fans out to scraper implementations, and the database stores normalized job records.

## 1. Startup sequence

```text
cmd/server/main.go
  -> db.InitDB()
  -> gin engine starts on :8080

cmd/app/main.go
  -> ui.Run()
  -> UI opens dashboard and settings screens
```

At startup:

- `db.InitDB()` creates the local SQLite schema if needed.
- The API server listens on `:8080` and exposes job/account routes.
- The Gio app launches and begins rendering the dashboard.
- The UI default state is the dashboard, with the API base URL stored in the app settings.

## 2. System components

### Database layer

`internal/db/database.go` owns persistence for:

- `Account`: `ID`, `Provider`, `Username`, `Enabled`
- `Job`: `ID`, `Title`, `URL`, `Category`, `Price`, `Description`, `Provider`, `CreatedAt`

Key functions:

- `InitDB()`
- `GetAccounts()`
- `AddAccount()`
- `GetEnabledAccounts()`
- `ClearJobs()`
- `SaveJob()`
- `GetJobs()`

The SQLite database is stored in `./jobs.db` and is created automatically when the server starts.

### API layer

`cmd/server/main.go` sets up the Gin service and implements these handlers:

- `GET /ping` — health check
- `GET /GetAccounts` — list saved accounts
- `POST /AddAccount` — create an account with provider + username
- `GET /GetJobs` — list all jobs
- `POST /StartCrawlJobs` — clear stale jobs and trigger scraping for each enabled account

The REST client in `internal/rest/app_rest.go` is used by the UI for every request.

### UI layer

`internal/ui/app.go` manages screens such as:

- `ScreenDashboard`
- `ScreenAddAccount`
- `ScreenSettings`

The dashboard renders:

- a Start All button
- an Add Account button
- a Settings button
- a filterable job list
- a status line for API connectivity and progress

## 3. "Start All" workflow

When the user clicks Start All in the dashboard, the UI calls the API through `client().StartCrawlJobs()`.

The server then performs the following:

1. `db.ClearJobs()` removes all existing rows from `jobs`
2. `db.GetEnabledAccounts()` loads all accounts with `enabled = 1`
3. For each enabled account, it calls `scraper.FetchJobs(account.Provider, account.Username)`
4. The selected provider returns a normalized list of `JobItem`
5. Each item is saved with `db.SaveJob(...)`
6. The API responds with `200 OK` once the crawl completes

## 4. Provider implementation details

`internal/scraper/provider.go` defines the shared registry contract:

```go
type JobProvider interface {
    Name() string
    FetchJobs(username string) ([]JobItem, error)
}
```

The provider model includes:

```go
type JobItem struct {
    Title       string
    URL         string
    Category    string
    Price       string
    Description string
}
```

The `FreelancerVietProvider` implementation is registered via `init()` and targets the Freelancer Việt listing page.

## 5. Data persistence rules

The job table deduplicates by URL:

```sql
INSERT OR IGNORE INTO jobs (title, url, category, price, description, provider)
VALUES (?, ?, ?, ?, ?, ?)
```

This prevents stale or repeated job ads from being saved multiple times during a refresh cycle.

## 6. Extending the app with a new provider

To add another provider, follow the same pattern as the existing implementation:

1. Create a spec document under `docs/providers/`
2. Implement a new type that satisfies `JobProvider`
3. Register it in an `init()` function
4. Rebuild the server and UI binaries
5. Add a saved account using the provider name in the UI

No database migration is required for a new provider because all records are normalized into the same schema before insertion.

## 7. End-to-end summary

```text
User opens UI
  -> UI pings the API
  -> User adds one or more accounts
  -> User clicks Start All
  -> UI POSTs /StartCrawlJobs
  -> Server clears old jobs
  -> Server loads enabled accounts
  -> Provider(s) scrape remote listings
  -> Jobs are normalized and saved to SQLite
  -> UI refreshes and shows the new table
```