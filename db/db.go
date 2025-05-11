package db

import (
	"bytes"
	"errors"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

var (
	defaultBucket  = []byte("default")
	replicaBucket  = []byte("replication")
)

// Database is an open bolt database.
type Database struct {
	db       *bolt.DB
	readOnly bool
}

// NewDatabase returns an instance of a database that we can work with.
func NewDatabase(dbPath string, readOnly bool) (*Database, func() error, error) {
	boltDb, err := bolt.Open(dbPath, 0o600, nil)
	if err != nil {
		return nil, nil, err
	}

	database := &Database{db: boltDb, readOnly: readOnly}
	closeFunc := func() error {
		return boltDb.Close()
	}

	if err := database.createBuckets(); err != nil {
		closeFunc()
		return nil, nil, fmt.Errorf("creating default bucket: %w", err)
	}

	return database, closeFunc, nil
}

func (d *Database) createBuckets() error {
	return d.db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(defaultBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(replicaBucket); err != nil {
			return err
		}
		return nil
	})
}

func (d *Database) SetKey(key string, value []byte) error {
	if d.readOnly {
		return errors.New("read-only mode")
	}

	return d.db.Update(func(tx *bolt.Tx) error {
		if err := tx.Bucket(defaultBucket).Put([]byte(key), value); err != nil {
			return err
		}
		return tx.Bucket(replicaBucket).Put([]byte(key), value)
	})
}

func (d *Database) SetKeyOnReplica(key string, value []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(defaultBucket).Put([]byte(key), value)
	})
}

func copyByteSlice(b []byte) []byte {
	if b == nil {
		return nil
	}
	res := make([]byte, len(b))
	copy(res, b)
	return res
}

func (d *Database) GetNextKeyForReplication() ([]byte, []byte, error) {
	var key, value []byte
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(replicaBucket)
		k, v := b.Cursor().First()
		key = copyByteSlice(k)
		value = copyByteSlice(v)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return key, value, nil
}

func (d *Database) DeleteReplicationKey(key, value []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(replicaBucket)
		v := b.Get(key)
		if v == nil {
			return errors.New("key does not exist")
		}
		if !bytes.Equal(v, value) {
			return errors.New("value does not match")
		}
		return b.Delete(key)
	})
}

func (d *Database) GetKey(key string) ([]byte, error) {
	var result []byte
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(defaultBucket)
		result = copyByteSlice(b.Get([]byte(key)))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (d *Database) DeleteExtraKeys(isExtra func(string) bool) error {
	var keys []string

	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(defaultBucket)
		return b.ForEach(func(k, v []byte) error {
			ks := string(k)
			if isExtra(ks) {
				keys = append(keys, ks)
			}
			return nil
		})
	})
	if err != nil {
		return err
	}

	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(defaultBucket)
		for _, k := range keys {
			if err := b.Delete([]byte(k)); err != nil {
				return err
			}
		}
		return nil
	})
}
