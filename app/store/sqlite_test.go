package store

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLite_Save(t *testing.T) {
	dbFile := "/tmp/test_sqlite_save.db"
	defer os.Remove(dbFile)

	s, err := NewSQLite(dbFile, time.Minute)
	require.NoError(t, err)
	defer s.Close()

	msg := Message{Key: "testkey123", Exp: time.Now().Add(time.Hour), Data: []byte("test data"), PinHash: "hash123", Errors: 0}
	err = s.Save(t.Context(), &msg)
	require.NoError(t, err)

	// verify by loading
	loaded, err := s.Load(t.Context(), "testkey123")
	require.NoError(t, err)
	assert.Equal(t, msg.Key, loaded.Key)
	assert.Equal(t, msg.Data, loaded.Data)
	assert.Equal(t, msg.PinHash, loaded.PinHash)
	assert.Equal(t, msg.Errors, loaded.Errors)
	assert.Equal(t, msg.Exp.Unix(), loaded.Exp.Unix())
}

func TestSQLite_Load(t *testing.T) {
	dbFile := "/tmp/test_sqlite_load.db"
	defer os.Remove(dbFile)

	s, err := NewSQLite(dbFile, time.Minute)
	require.NoError(t, err)
	defer s.Close()

	original := Message{Key: "roundtrip", Exp: time.Now().Add(time.Hour), Data: []byte("round trip data"), PinHash: "pinpin", Errors: 5}
	require.NoError(t, s.Save(t.Context(), &original))

	loaded, err := s.Load(t.Context(), "roundtrip")
	require.NoError(t, err)
	assert.Equal(t, original.Key, loaded.Key)
	assert.Equal(t, original.Data, loaded.Data)
	assert.Equal(t, original.PinHash, loaded.PinHash)
	assert.Equal(t, original.Errors, loaded.Errors)
	assert.Equal(t, original.Exp.Unix(), loaded.Exp.Unix())
}

func TestSQLite_Load_NotFound(t *testing.T) {
	dbFile := "/tmp/test_sqlite_notfound.db"
	defer os.Remove(dbFile)

	s, err := NewSQLite(dbFile, time.Minute)
	require.NoError(t, err)
	defer s.Close()

	_, err = s.Load(t.Context(), "nonexistent")
	assert.Equal(t, ErrLoadRejected, err)
}

func TestSQLite_IncErr(t *testing.T) {
	dbFile := "/tmp/test_sqlite_incerr.db"
	defer os.Remove(dbFile)

	s, err := NewSQLite(dbFile, time.Minute)
	require.NoError(t, err)
	defer s.Close()

	msg := Message{Key: "errkey", Exp: time.Now().Add(time.Hour), Data: []byte("data"), PinHash: "hash"}
	require.NoError(t, s.Save(t.Context(), &msg))

	cnt, err := s.IncErr(t.Context(), "errkey")
	require.NoError(t, err)
	assert.Equal(t, 1, cnt)

	cnt, err = s.IncErr(t.Context(), "errkey")
	require.NoError(t, err)
	assert.Equal(t, 2, cnt)

	cnt, err = s.IncErr(t.Context(), "errkey")
	require.NoError(t, err)
	assert.Equal(t, 3, cnt)

	// non-existent key
	_, err = s.IncErr(t.Context(), "nokey")
	assert.Equal(t, ErrLoadRejected, err)
}

func TestSQLite_IncErr_Concurrent(t *testing.T) {
	dbFile := "/tmp/test_sqlite_concurrent.db"
	defer os.Remove(dbFile)

	s, err := NewSQLite(dbFile, time.Minute)
	require.NoError(t, err)
	defer s.Close()

	ctx := t.Context()
	msg := Message{Key: "conckey", Exp: time.Now().Add(time.Hour), Data: []byte("data"), PinHash: "hash"}
	require.NoError(t, s.Save(ctx, &msg))

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			_, _ = s.IncErr(ctx, "conckey")
		}()
	}
	wg.Wait()

	// load and verify final count equals numGoroutines
	loaded, err := s.Load(ctx, "conckey")
	require.NoError(t, err)
	assert.Equal(t, numGoroutines, loaded.Errors, "should have exactly %d errors after concurrent increments", numGoroutines)
}

func TestSQLite_Remove(t *testing.T) {
	dbFile := "/tmp/test_sqlite_remove.db"
	defer os.Remove(dbFile)

	s, err := NewSQLite(dbFile, time.Minute)
	require.NoError(t, err)
	defer s.Close()

	msg := Message{Key: "toremove", Exp: time.Now().Add(time.Hour), Data: []byte("data"), PinHash: "hash"}
	require.NoError(t, s.Save(t.Context(), &msg))

	// verify it exists
	_, err = s.Load(t.Context(), "toremove")
	require.NoError(t, err)

	// remove
	err = s.Remove(t.Context(), "toremove")
	require.NoError(t, err)

	// verify it's gone
	_, err = s.Load(t.Context(), "toremove")
	assert.Equal(t, ErrLoadRejected, err)
}

func TestSQLite_Cleanup(t *testing.T) {
	dbFile := "/tmp/test_sqlite_cleanup.db"
	defer os.Remove(dbFile)

	// use short cleanup duration for test
	s, err := NewSQLite(dbFile, time.Millisecond*50)
	require.NoError(t, err)
	defer s.Close()

	// create expired message (exp in past)
	msg := Message{Key: "expired", Exp: time.Now().Add(-time.Hour), Data: []byte("old data"), PinHash: "hash"}
	require.NoError(t, s.Save(t.Context(), &msg))

	// verify it was saved
	_, err = s.Load(t.Context(), "expired")
	require.NoError(t, err)

	// wait for cleaner to run
	time.Sleep(time.Millisecond * 150)

	// verify it's gone
	_, err = s.Load(t.Context(), "expired")
	assert.Equal(t, ErrLoadRejected, err)
}

func TestSQLite_Cleanup_KeepsValid(t *testing.T) {
	dbFile := "/tmp/test_sqlite_cleanup_valid.db"
	defer os.Remove(dbFile)

	s, err := NewSQLite(dbFile, time.Millisecond*50)
	require.NoError(t, err)
	defer s.Close()

	// create message with future expiry
	msg := Message{Key: "valid", Exp: time.Now().Add(time.Hour), Data: []byte("fresh data"), PinHash: "hash"}
	require.NoError(t, s.Save(t.Context(), &msg))

	// wait for cleaner to run
	time.Sleep(time.Millisecond * 150)

	// verify it still exists
	loaded, err := s.Load(t.Context(), "valid")
	require.NoError(t, err)
	assert.Equal(t, msg.Key, loaded.Key)
}

func TestSQLite_CleanerStopsOnClose(t *testing.T) {
	s := NewInMemory(time.Millisecond * 10) // fast cleanup interval

	// save a message that will expire
	msg := Message{Key: "willexpire", Exp: time.Now().Add(-time.Hour), Data: []byte("old"), PinHash: "hash"}
	require.NoError(t, s.Save(t.Context(), &msg))

	// close immediately
	require.NoError(t, s.Close())

	// wait longer than cleanup interval - if goroutine still runs, it will panic or log errors
	time.Sleep(time.Millisecond * 50)

	// if we get here without panic, cleaner stopped properly
	// the test would have logged errors like "cleanup failed: sql: database is closed" if not fixed
}

func TestInMemory_SharedAcrossConnections(t *testing.T) {
	// this test verifies that in-memory SQLite shares data across all operations
	// with :memory: and connection pooling, each connection gets isolated DB - this is a bug
	s := NewInMemory(time.Minute)
	defer s.Close()

	ctx := t.Context()
	// save a message
	msg := Message{Key: "memtest", Exp: time.Now().Add(time.Hour), Data: []byte("test data"), PinHash: "hash"}
	require.NoError(t, s.Save(ctx, &msg))

	// force new connection by doing multiple concurrent loads
	// if connections are isolated, some loads will fail
	const iterations = 10
	var wg sync.WaitGroup
	wg.Add(iterations)
	errors := make(chan error, iterations)

	for range iterations {
		go func() {
			defer wg.Done()
			_, err := s.Load(ctx, "memtest")
			if err != nil {
				errors <- err
			}
		}()
	}
	wg.Wait()
	close(errors)

	// all loads should succeed - if any fail, the DB isn't shared
	loadErrors := make([]error, 0, iterations)
	for err := range errors {
		loadErrors = append(loadErrors, err)
	}
	assert.Empty(t, loadErrors, "all concurrent loads should succeed with shared in-memory DB")
}
