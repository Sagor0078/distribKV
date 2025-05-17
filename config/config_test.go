package config_test

import (
	"os"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/Sagor0078/distribKV/config"
)

const tomlData = `
[[shards]]
name = "shard-0"
idx = 0
address = "127.0.0.2:8080"
replicas = ["127.0.0.22:8080", "127.0.0.23:8080"]

[[shards]]
name = "shard-1"
idx = 1
address = "127.0.0.3:8080"
replicas = ["127.0.0.33:8080"]
`

func writeTempTOML(t *testing.T) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "sharding-*.toml")
	assert.NoError(t, err)

	_, err = tmpFile.WriteString(tomlData)
	assert.NoError(t, err)

	err = tmpFile.Close()
	assert.NoError(t, err)

	return tmpFile.Name()
}

func TestParseFile(t *testing.T) {
	filename := writeTempTOML(t)
	defer os.Remove(filename)

	cfg, err := config.ParseFile(filename)
	assert.NoError(t, err)
	assert.Len(t, cfg.Shards, 2)
	assert.Equal(t, "shard-0", cfg.Shards[0].Name)
}

func TestParseShards(t *testing.T) {
	shards := []config.Shard{
		{
			Name:     "shard-0",
			Idx:      0,
			Address:  "127.0.0.2:8080",
			Replicas: []string{"127.0.0.22:8080"},
		},
		{
			Name:     "shard-1",
			Idx:      1,
			Address:  "127.0.0.3:8080",
			Replicas: []string{"127.0.0.33:8080"},
		},
	}

	s, err := config.ParseShards(shards, "shard-0")
	assert.NoError(t, err)
	assert.Equal(t, 2, s.Count)
	assert.Equal(t, 0, s.CurIdx)
	assert.Equal(t, "127.0.0.2:8080", s.Addrs[0])
	assert.Equal(t, "127.0.0.3:8080", s.Addrs[1])
	assert.Equal(t, []string{"127.0.0.22:8080"}, s.Replicas[0])
}

func TestIndex(t *testing.T) {
	s := &config.Shards{
		Count: 4,
	}
	index := s.Index("test-key")
	assert.True(t, index >= 0 && index < 4)
}

func TestGetReplicas(t *testing.T) {
	s := &config.Shards{
		Replicas: map[int][]string{
			0: {"127.0.0.22:8080"},
			1: {"127.0.0.33:8080"},
		},
	}
	assert.Equal(t, []string{"127.0.0.22:8080"}, s.GetReplicas(0))
	assert.Equal(t, []string{"127.0.0.33:8080"}, s.GetReplicas(1))
}
