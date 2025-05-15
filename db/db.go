package db

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v4"
)

var (
	replicaPrefix = []byte("replica:")
)

// Database wraps a Badger DB instance.
type Database struct {
	db       *badger.DB
	readOnly bool
}

// NewDatabase initializes and returns a new Badger database.
// It ensures the dbPath exists before opening, to prevent "no manifest found" errors in read-only mode.
func NewDatabase(dbPath string, readOnly bool) (*Database, func() error, error) {
	// Ensure directory exists
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create DB directory %q: %w", dbPath, err)
	}

	opts := badger.DefaultOptions(dbPath).WithReadOnly(false)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, nil, err
	}

	closeFunc := func() error {
		return db.Close()
	}

	return &Database{db: db, readOnly: readOnly}, closeFunc, nil
}

// BootstrapReplica copies all files from srcDBPath to replicaDBPath.
// This is useful to initialize replicas before opening them in read-only mode.
func BootstrapReplica(srcDBPath, replicaDBPath string) error {
	// Ensure target dir exists
	if err := os.MkdirAll(replicaDBPath, 0755); err != nil {
		return fmt.Errorf("failed to create replica directory %q: %w", replicaDBPath, err)
	}

	srcFiles, err := os.ReadDir(srcDBPath)
	if err != nil {
		return fmt.Errorf("failed to read source DB directory %q: %w", srcDBPath, err)
	}

	for _, file := range srcFiles {
		srcFile := filepath.Join(srcDBPath, file.Name())
		dstFile := filepath.Join(replicaDBPath, file.Name())

		// Skip if destination file already exists
		if _, err := os.Stat(dstFile); err == nil {
			continue
		}

		if err := copyFile(srcFile, dstFile); err != nil {
			return fmt.Errorf("failed to copy %q to %q: %w", srcFile, dstFile, err)
		}
	}
	return nil
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Sync()
		_ = out.Close()
	}()

	_, err = io.Copy(out, in)
	return err
}

// prefixKey adds a prefix for replication keys
func prefixKey(prefix, key []byte) []byte {
	return append(prefix, key...)
}

// SetKey writes a key to the main store and the replication queue.
func (d *Database) SetKey(key string, value []byte) error {
	if d.readOnly {
		return errors.New("read-only mode")
	}

	return d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set([]byte(key), value); err != nil {
			return err
		}
		return txn.Set(prefixKey(replicaPrefix, []byte(key)), value)
	})
}

// SetKeyOnReplica writes a key directly to the main store (used by replicas).
func (d *Database) SetKeyOnReplica(key string, value []byte) error {
	if d.readOnly {
		return errors.New("read-only mode")
	}
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

// GetNextKeyForReplication fetches the first replication entry.
func (d *Database) GetNextKeyForReplication() ([]byte, []byte, error) {
	var k, v []byte
	err := d.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(replicaPrefix); it.ValidForPrefix(replicaPrefix); it.Next() {
			item := it.Item()
			key := item.KeyCopy(nil)
			err := item.Value(func(val []byte) error {
				k = key[len(replicaPrefix):] // Strip prefix
				v = append([]byte{}, val...)
				return nil
			})
			return err
		}
		return nil
	})

	if err != nil {
		return nil, nil, err
	}
	return k, v, nil
}

// DeleteReplicationKey deletes a key from the replication queue if the value matches.
func (d *Database) DeleteReplicationKey(key, value []byte) error {
	if d.readOnly {
		return errors.New("read-only mode")
	}

	prefixedKey := prefixKey(replicaPrefix, key)
	return d.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(prefixedKey)
		if err != nil {
			return err
		}

		var actual []byte
		err = item.Value(func(val []byte) error {
			actual = val
			return nil
		})
		if err != nil {
			return err
		}

		if !bytes.Equal(actual, value) {
			return fmt.Errorf("value mismatch for key %s: expected %s, got %s", key, value, actual)
		}

		// Proceed with deletion
		return txn.Delete(prefixedKey)
	})
}

// GetKey retrieves a key's value from the main store.
func (d *Database) GetKey(key string) ([]byte, error) {
	var result []byte
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			result = append([]byte{}, val...)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteExtraKeys removes keys that don't belong to this shard.
func (d *Database) DeleteExtraKeys(isExtra func(string) bool) error {
	if d.readOnly {
		return errors.New("read-only mode")
	}

	var keysToDelete [][]byte

	err := d.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			key := it.Item().Key()
			if bytes.HasPrefix(key, replicaPrefix) {
				continue // skip replica entries
			}
			kStr := string(key)
			if isExtra(kStr) {
				keysToDelete = append(keysToDelete, append([]byte{}, key...))
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return d.db.Update(func(txn *badger.Txn) error {
		for _, k := range keysToDelete {
			if err := txn.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}
