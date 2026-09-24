package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type item struct {
	FileID            int64
	Path, ETag, Stage string
	Size              int64
	ModTime           time.Time
	Tags              []string
}
type queueStore struct{ db *sql.DB }

func openStore(path string) (*queueStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, q := range []string{
		`PRAGMA journal_mode=WAL`, `PRAGMA synchronous=FULL`, `PRAGMA busy_timeout=10000`,
		`CREATE TABLE IF NOT EXISTS queue (file_id INTEGER PRIMARY KEY, path TEXT NOT NULL, etag TEXT NOT NULL, size INTEGER NOT NULL, mtime_ns INTEGER NOT NULL, stage TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS state (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(q); err != nil {
			db.Close()
			return nil, fmt.Errorf("initialize SQLite: %w", err)
		}
	}
	return &queueStore{db: db}, nil
}

func (s *queueStore) Close() error { return s.db.Close() }
func (s *queueStore) enqueue(i item) error {
	if i.Stage == "" {
		i.Stage = "pending"
	}
	_, err := s.db.Exec(`INSERT INTO queue(file_id,path,etag,size,mtime_ns,stage,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(file_id) DO UPDATE SET path=excluded.path,etag=excluded.etag,size=excluded.size,mtime_ns=excluded.mtime_ns,stage=CASE WHEN queue.stage='deferred' THEN 'pending' ELSE queue.stage END,updated_at=excluded.updated_at`, i.FileID, i.Path, i.ETag, i.Size, i.ModTime.UnixNano(), i.Stage, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *queueStore) recover() error {
	if _, err := s.db.Exec(`DELETE FROM queue WHERE stage='tagged'`); err != nil {
		return err
	}
	_, err := s.db.Exec(`UPDATE queue SET stage='pending',updated_at=? WHERE stage<>'pending'`, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *queueStore) pending() ([]item, error) {
	rows, err := s.db.Query(`SELECT file_id,path,etag,size,mtime_ns,stage FROM queue WHERE stage='pending' ORDER BY file_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []item
	for rows.Next() {
		var i item
		var ns int64
		if err := rows.Scan(&i.FileID, &i.Path, &i.ETag, &i.Size, &ns, &i.Stage); err != nil {
			return nil, err
		}
		i.ModTime = time.Unix(0, ns)
		out = append(out, i)
	}
	return out, rows.Err()
}
func (s *queueStore) setStage(id int64, stage string) error {
	_, err := s.db.Exec(`UPDATE queue SET stage=?,updated_at=? WHERE file_id=?`, stage, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}
func (s *queueStore) refresh(oldID int64, i item) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if oldID != i.FileID {
		if _, err = tx.Exec(`DELETE FROM queue WHERE file_id=?`, oldID); err != nil {
			return err
		}
	}
	_, err = tx.Exec(`INSERT INTO queue(file_id,path,etag,size,mtime_ns,stage,updated_at) VALUES(?,?,?,?,?,'pending',?) ON CONFLICT(file_id) DO UPDATE SET path=excluded.path,etag=excluded.etag,size=excluded.size,mtime_ns=excluded.mtime_ns,stage='pending',updated_at=excluded.updated_at`, i.FileID, i.Path, i.ETag, i.Size, i.ModTime.UnixNano(), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (s *queueStore) remove(id int64) error {
	_, err := s.db.Exec(`DELETE FROM queue WHERE file_id=?`, id)
	return err
}
func (s *queueStore) count() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM queue`).Scan(&n)
	return n, err
}
func (s *queueStore) setState(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO state(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}
func (s *queueStore) getState(key string) (time.Time, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM state WHERE key=?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, value)
}
func (s *queueStore) startRun(t time.Time) error {
	if err := s.setState("last_run_started_at", t.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return s.setState("last_run_status", "running")
}
func (s *queueStore) finishRun(t time.Time) error {
	if err := s.setState("last_successful_run_at", t.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return s.setState("last_run_status", "success")
}
func (s *queueStore) checkpoints() (time.Time, time.Time, error) {
	a, e := s.getState("last_run_started_at")
	if e != nil {
		return a, time.Time{}, e
	}
	b, e := s.getState("last_successful_run_at")
	return a, b, e
}
