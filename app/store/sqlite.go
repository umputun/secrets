package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/go-pkgz/lgr"
	_ "modernc.org/sqlite" // sqlite driver
)

// SQLite implements store.Engine with SQLite database
type SQLite struct {
	db   *sql.DB
	lock sync.Mutex
	done chan struct{}
}

// NewInMemory creates an ephemeral in-memory SQLite store.
// Each call creates an isolated database using a unique URI.
func NewInMemory(cleanupDuration time.Duration) *SQLite {
	// generate unique URI to isolate each in-memory store instance
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic("failed to generate random bytes: " + err.Error())
	}
	uri := "file:" + hex.EncodeToString(buf[:]) + "?mode=memory&cache=shared"
	s, err := NewSQLite(uri, cleanupDuration)
	if err != nil {
		panic("failed to create in-memory sqlite: " + err.Error())
	}
	return s
}

// NewSQLite creates a persistent SQLite-based store
func NewSQLite(dbFile string, cleanupDuration time.Duration) (*SQLite, error) {
	log.Printf("[INFO] sqlite (%s) store", dbFile)

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	ctx := context.Background()

	// configure sqlite for better performance
	if _, err = db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err = db.ExecContext(ctx, "PRAGMA synchronous=NORMAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set synchronous mode: %w", err)
	}

	// create table and index
	schema := `
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			exp INTEGER NOT NULL,
			data BLOB NOT NULL,
			pin_hash TEXT NOT NULL,
			errors INTEGER DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_messages_exp ON messages(exp);
	`
	if _, err = db.ExecContext(ctx, schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	result := &SQLite{db: db, done: make(chan struct{})}
	result.activateCleaner(cleanupDuration)
	return result, nil
}

// Save stores message in the database
func (s *SQLite) Save(ctx context.Context, msg *Message) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO messages (id, exp, data, pin_hash, errors) VALUES (?, ?, ?, ?, ?)",
		msg.Key, msg.Exp.Unix(), msg.Data, msg.PinHash, msg.Errors,
	)
	if err != nil {
		log.Printf("[ERROR] failed to save message: %v", err)
		return ErrSaveRejected
	}
	log.Printf("[DEBUG] saved, exp=%v", msg.Exp.Local().Format(time.RFC3339))
	return nil
}

// Load retrieves message by key
func (s *SQLite) Load(ctx context.Context, key string) (*Message, error) {
	var msg Message
	var expUnix int64

	err := s.db.QueryRowContext(ctx,
		"SELECT id, exp, data, pin_hash, errors FROM messages WHERE id = ?",
		key,
	).Scan(&msg.Key, &expUnix, &msg.Data, &msg.PinHash, &msg.Errors)

	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("[DEBUG] not found %s", key)
		return nil, ErrLoadRejected
	}
	if err != nil {
		log.Printf("[ERROR] failed to load message: %v", err)
		return nil, ErrLoadRejected
	}

	msg.Exp = time.Unix(expUnix, 0)
	return &msg, nil
}

// IncErr atomically increments the error count and returns new value
func (s *SQLite) IncErr(ctx context.Context, key string) (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var count int
	err := s.db.QueryRowContext(ctx,
		"UPDATE messages SET errors = errors + 1 WHERE id = ? RETURNING errors",
		key,
	).Scan(&count)

	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("[DEBUG] not found %s", key)
		return 0, ErrLoadRejected
	}
	if err != nil {
		log.Printf("[ERROR] failed to increment errors: %v", err)
		return 0, ErrLoadRejected
	}

	return count, nil
}

// Remove deletes message by key
func (s *SQLite) Remove(ctx context.Context, key string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM messages WHERE id = ?", key)
	if err != nil {
		log.Printf("[ERROR] failed to remove message: %v", err)
		return fmt.Errorf("remove message: %w", err)
	}
	log.Printf("[INFO] removed %s", key)
	return nil
}

// Close closes the database connection and stops the cleaner goroutine
func (s *SQLite) Close() error {
	close(s.done)
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close sqlite: %w", err)
	}
	return nil
}

// activateCleaner runs periodic cleanup of expired messages
func (s *SQLite) activateCleaner(every time.Duration) {
	log.Printf("[INFO] cleaner activated, every %v", every)

	ticker := time.NewTicker(every)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				s.lock.Lock()
				result, err := s.db.ExecContext(context.Background(), "DELETE FROM messages WHERE exp < ?", time.Now().Unix())
				s.lock.Unlock()
				if err != nil {
					log.Printf("[WARN] cleanup failed: %v", err)
					continue
				}
				if count, _ := result.RowsAffected(); count > 0 {
					log.Printf("[INFO] cleaned %d expired messages", count)
				}
			}
		}
	}()
}
