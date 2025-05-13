package replication

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/Sagor0078/distribKV/db"
)

type NextKeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Err   error  `json:"-"`
}

type client struct {
	db         *db.Database
	leaderAddr string
}

func ClientLoop(db *db.Database, leaderAddr string) {
	c := &client{db: db, leaderAddr: leaderAddr}
	for {
		ok, err := c.loop()
		if err != nil {
			log.Printf("Replication loop error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if !ok {
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func (c *client) loop() (bool, error) {
	resp, err := http.Get("http://" + c.leaderAddr + "/next-replication-key")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var res NextKeyValue
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return false, err
	}

	if res.Key == "" {
		return false, nil
	}

	if err := c.db.SetKeyOnReplica(res.Key, []byte(res.Value)); err != nil {
		return false, err
	}

	if err := c.deleteFromReplicationQueue(res.Key, res.Value); err != nil {
		log.Printf("Failed to delete replication key: %v", err)
	}

	return true, nil
}

func (c *client) deleteFromReplicationQueue(key, value string) error {
	u := url.Values{}
	u.Set("key", key)
	u.Set("value", value)

	resp, err := http.Get("http://" + c.leaderAddr + "/delete-replication-key?" + u.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, []byte("ok")) {
		return errors.New(string(data))
	}
	return nil
}
