#!/bin/bash
set -e
trap 'killall distribKV' SIGINT

# Build binary
go install -v

# Kill old processes if any
killall distribkv || true
sleep 0.1

# Shard-0
distribkv -db-location=shard-0.db -http-addr=127.0.0.2:8080 -config-file=sharding.toml -shard=shard-0 &
distribkv -db-location=shard-0-r1.db -http-addr=127.0.0.22:8080 -config-file=sharding.toml -shard=shard-0 -replica &
distribkv -db-location=shard-0-r2.db -http-addr=127.0.0.23:8080 -config-file=sharding.toml -shard=shard-0 -replica &

# Shard-1
distribkv -db-location=shard-1.db -http-addr=127.0.0.3:8080 -config-file=sharding.toml -shard=shard-1 &
distribkv -db-location=shard-1-r1.db -http-addr=127.0.0.33:8080 -config-file=sharding.toml -shard=shard-1 -replica &

# Shard-2
distribkv -db-location=shard-2.db -http-addr=127.0.0.4:8080 -config-file=sharding.toml -shard=shard-2 &
distribkv -db-location=shard-2-r1.db -http-addr=127.0.0.44:8080 -config-file=sharding.toml -shard=shard-2 -replica &
distribkv -db-location=shard-2-r2.db -http-addr=127.0.0.45:8080 -config-file=sharding.toml -shard=shard-2 -replica &

# Shard-3
distribkv -db-location=shard-3.db -http-addr=127.0.0.5:8080 -config-file=sharding.toml -shard=shard-3 &
distribkv -db-location=shard-3-r1.db -http-addr=127.0.0.55:8080 -config-file=sharding.toml -shard=shard-3 -replica &

wait
