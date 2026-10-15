package ui

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	"gioui.org/app"
	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("setting not found")

type Store struct {
	db *sql.DB
}

func NewStore() (*Store, error) {
	// Lấy thư mục data chuẩn của Gio (tự động xử lý Android, Desktop, iOS)
	dir, err := app.DataDir()
	if err != nil {
		// Fallback an toàn nếu gọi trước khi app khởi tạo hoàn toàn
		dir = "."
	}

	dbPath := filepath.Join(dir, "data.db")

	// Đảm bảo thư mục cha tồn tại
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Save(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings(key, value) VALUES(?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

func (s *Store) Get(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return value, err
}

func (s *Store) Close() error { return s.db.Close() }
