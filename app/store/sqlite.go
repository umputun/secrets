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
	db      *sql.DB
	lock    sync.RWMutex
	done    chan struct{}
	cleanWg sync.WaitGroup
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

	// configure connection pool for SQLite (single writer)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

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
	if _, err = db.ExecContext(ctx, "PRAGMA busy_timeout=5000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}

	// create table and index
	schema := `
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			exp INTEGER NOT NULL,
			data BLOB NOT NULL,
			pin_hash TEXT NOT NULL,
			errors INTEGER DEFAULT 0,
			client_enc INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_messages_exp ON messages(exp);
	`
	if _, err = db.ExecContext(ctx, schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	// migrate existing databases: add client_enc column if missing
	if err = migrateClientEnc(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate client_enc: %w", err)
	}

	result := &SQLite{db: db, done: make(chan struct{})}
	result.activateCleaner(cleanupDuration)
	return result, nil
}

// Save stores message in the database
func (s *SQLite) Save(ctx context.Context, msg *Message) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	clientEnc := 0
	if msg.ClientEnc {
		clientEnc = 1
	}

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO messages (id, exp, data, pin_hash, errors, client_enc) VALUES (?, ?, ?, ?, ?, ?)",
		msg.Key, msg.Exp.Unix(), msg.Data, msg.PinHash, msg.Errors, clientEnc,
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
	s.lock.RLock()
	defer s.lock.RUnlock()

	var msg Message
	var expUnix int64
	var clientEnc int

	err := s.db.QueryRowContext(ctx,
		"SELECT id, exp, data, pin_hash, errors, client_enc FROM messages WHERE id = ?",
		key,
	).Scan(&msg.Key, &expUnix, &msg.Data, &msg.PinHash, &msg.Errors, &clientEnc)

	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("[DEBUG] not found %s", key)
		return nil, ErrLoadRejected
	}
	if err != nil {
		log.Printf("[ERROR] failed to load message: %v", err)
		return nil, ErrLoadRejected
	}

	msg.Exp = time.Unix(expUnix, 0)
	msg.ClientEnc = clientEnc != 0
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
	s.cleanWg.Wait() // wait for cleaner goroutine to finish
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close sqlite: %w", err)
	}
	return nil
}

// activateCleaner runs periodic cleanup of expired messages
func (s *SQLite) activateCleaner(every time.Duration) {
	log.Printf("[INFO] cleaner activated, every %v", every)

	s.cleanWg.Add(1)
	ticker := time.NewTicker(every)
	go func() {
		defer s.cleanWg.Done()
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

// migrateClientEnc adds client_enc column to existing databases
func migrateClientEnc(ctx context.Context, db *sql.DB) error {
	// check if column exists using PRAGMA table_info
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(messages)")
	if err != nil {
		return fmt.Errorf("query table info: %w", err)
	}
	defer rows.Close()

	hasClientEnc := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("scan table info: %w", err)
		}
		if name == "client_enc" {
			hasClientEnc = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table info: %w", err)
	}

	if !hasClientEnc {
		log.Printf("[INFO] migrating database: adding client_enc column")
		_, err := db.ExecContext(ctx, "ALTER TABLE messages ADD COLUMN client_enc INTEGER NOT NULL DEFAULT 0")
		if err != nil {
			return fmt.Errorf("add client_enc column: %w", err)
		}
	}
	return nil
}
