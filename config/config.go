package config

import (
	"fmt"
	"hash/fnv"
	"os"

	"github.com/BurntSushi/toml"
)

// Shard defines a node in the cluster with replicas.
type Shard struct {
	Name     string   `toml:"name"`
	Idx      int      `toml:"idx"`
	Address  string   `toml:"address"`
	Replicas []string `toml:"replicas"`
}

// Config holds the list of shards.
type Config struct {
	Shards []Shard `toml:"shards"`
}

// ParseFile parses the TOML file into a Config.
func ParseFile(filename string) (Config, error) {
	var c Config
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("error reading file: %w", err)
	}
	if _, err := toml.Decode(string(data), &c); err != nil {
		return Config{}, fmt.Errorf("TOML decode error: %w", err)
	}
	return c, nil
}

// Shards holds parsed shard metadata for routing.
type Shards struct {
	Count    int
	CurIdx   int
	Addrs    map[int]string
	Replicas map[int][]string
}

// ParseShards validates and converts shard configuration.
func ParseShards(shards []Shard, curShardName string) (*Shards, error) {
	count := len(shards)
	addrs := make(map[int]string)
	replicas := make(map[int][]string)
	curIdx := -1

	for _, s := range shards {
		if _, exists := addrs[s.Idx]; exists {
			return nil, fmt.Errorf("duplicate shard index: %d", s.Idx)
		}
		addrs[s.Idx] = s.Address
		replicas[s.Idx] = s.Replicas

		if s.Name == curShardName {
			curIdx = s.Idx
		}
	}

	for i := 0; i < count; i++ {
		if _, ok := addrs[i]; !ok {
			return nil, fmt.Errorf("missing shard with index: %d", i)
		}
	}

	if curIdx == -1 {
		return nil, fmt.Errorf("current shard %q not found", curShardName)
	}

	return &Shards{
		Count:    count,
		CurIdx:   curIdx,
		Addrs:    addrs,
		Replicas: replicas,
	}, nil
}

// Index determines the shard index for a given key.
func (s *Shards) Index(key string) int {
	h := fnv.New64()
	_, _ = h.Write([]byte(key))
	return int(h.Sum64() % uint64(s.Count))
}

// GetReplicas returns the replicas for a given shard index.
func (s *Shards) GetReplicas(idx int) []string {
	return s.Replicas[idx]
}
