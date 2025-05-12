
// test for locally setup badger db
package main

import (
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v4"
)

func main() {
	// Open BadgerDB in a local directory
	opts := badger.DefaultOptions("./badgerdb").WithLoggingLevel(badger.INFO)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Set a key
	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("name"), []byte("BadgerDB Local Test"))
	})
	if err != nil {
		log.Fatal(err)
	}

	// Get the key
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("name"))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			fmt.Printf("name: %s\n", val)
			return nil
		})
	})
	if err != nil {
		log.Fatal(err)
	}
}
