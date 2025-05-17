package db_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/Sagor0078/distribKV/db"
)

func createTempDir(t *testing.T) string {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(dir, 0755))
	return dir
}

func TestDatabase_SetGetKey(t *testing.T) {
	dir := createTempDir(t)
	dbInstance, closeFunc, err := db.NewDatabase(dir, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, closeFunc()) })

	key := "foo"
	value := []byte("bar")
	require.NoError(t, dbInstance.SetKey(key, value))

	val, err := dbInstance.GetKey(key)
	require.NoError(t, err)
	require.Equal(t, value, val)
}

func TestDatabase_ReplicationQueue(t *testing.T) {
	dir := createTempDir(t)
	dbInstance, closeFunc, err := db.NewDatabase(dir, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, closeFunc()) })

	key := "replicated-key"
	value := []byte("replicated-value")
	require.NoError(t, dbInstance.SetKey(key, value))

	k, v, err := dbInstance.GetNextKeyForReplication()
	require.NoError(t, err)
	require.Equal(t, key, string(k))
	require.Equal(t, value, v)

	require.NoError(t, dbInstance.DeleteReplicationKey([]byte(k), v))

	// Try to fetch again, should be nothing
	k, v, err = dbInstance.GetNextKeyForReplication()
	require.NoError(t, err)
	require.Nil(t, k)
	require.Nil(t, v)
}

func TestDatabase_SetKeyOnReplica(t *testing.T) {
	dir := createTempDir(t)
	dbInstance, closeFunc, err := db.NewDatabase(dir, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, closeFunc()) })

	key := "replica-key"
	val := []byte("replica-value")
	require.NoError(t, dbInstance.SetKeyOnReplica(key, val))

	fetched, err := dbInstance.GetKey(key)
	require.NoError(t, err)
	require.Equal(t, val, fetched)
}

func TestDatabase_DeleteExtraKeys(t *testing.T) {
	dir := createTempDir(t)
	dbInstance, closeFunc, err := db.NewDatabase(dir, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, closeFunc()) })

	require.NoError(t, dbInstance.SetKey("keep-key", []byte("1")))
	require.NoError(t, dbInstance.SetKey("remove-key", []byte("2")))

	require.NoError(t, dbInstance.DeleteExtraKeys(func(key string) bool {
		return key == "remove-key"
	}))

	val, err := dbInstance.GetKey("remove-key")
	require.Error(t, err)
	require.Nil(t, val)

	val, err = dbInstance.GetKey("keep-key")
	require.NoError(t, err)
	require.Equal(t, []byte("1"), val)
}

func TestDatabase_BootstrapReplica(t *testing.T) {
	src := createTempDir(t)
	dst := createTempDir(t)

	dbInstance, closeFunc, err := db.NewDatabase(src, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, closeFunc()) })

	key := "hello"
	val := []byte("world")
	require.NoError(t, dbInstance.SetKey(key, val))

	require.NoError(t, closeFunc())
	require.NoError(t, db.BootstrapReplica(src, dst))

	replica, replicaClose, err := db.NewDatabase(dst, true)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, replicaClose()) })

	fetched, err := replica.GetKey(key)
	require.NoError(t, err)
	require.Equal(t, val, fetched)
}
