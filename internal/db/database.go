package db

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

type Account struct {
	ID       int
	Provider string
	Username string
	Enabled  bool
}

type Job struct {
	ID        int
	Title     string
	URL       string
	Category  string
	Price     string
	Provider  string
	CreatedAt string
}

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite", "./jobs.db")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	createTables := `
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
	`
	_, err = DB.Exec(createTables)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}
}

func GetAccounts() []Account {
	rows, err := DB.Query("SELECT id, provider, username, enabled FROM accounts")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		rows.Scan(&a.ID, &a.Provider, &a.Username, &a.Enabled)
		accounts = append(accounts, a)
	}
	return accounts
}

func AddAccount(provider, username string) {
	DB.Exec("INSERT INTO accounts (provider, username, enabled) VALUES (?, ?, ?)", provider, username, true)
}

func ClearJobs() {
	_, err := DB.Exec("DELETE FROM jobs")
	if err != nil {
		log.Printf("Failed to clear jobs: %v", err)
	}
}

func SaveJob(title, url, category, price, provider string) {
	_, err := DB.Exec("INSERT OR IGNORE INTO jobs (title, url, category, price, provider) VALUES (?, ?, ?, ?, ?)", title, url, category, price, provider)
	if err != nil {
		log.Printf("Failed to save job: %v", err)
	}
}

func GetJobs() []Job {
	rows, err := DB.Query("SELECT id, title, url, category, price, provider, created_at FROM jobs ORDER BY id DESC")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		rows.Scan(&j.ID, &j.Title, &j.URL, &j.Category, &j.Price, &j.Provider, &j.CreatedAt)
		jobs = append(jobs, j)
	}
	return jobs
}