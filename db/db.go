package db

import (
	"bytes"
	"errors"
	"fmt"

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
func NewDatabase(dbPath string, readOnly bool) (*Database, func() error, error) {
	opts := badger.DefaultOptions(dbPath).WithReadOnly(readOnly)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, nil, err
	}

	closeFunc := func() error {
		return db.Close()
	}

	return &Database{db: db, readOnly: readOnly}, closeFunc, nil
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
