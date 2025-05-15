#!/bin/bash
set -e

# Clean up old DBs 
rm -rf shard-*.db

# Build binary
go build -o distribKV

# Start shards
./distribKV -db-location=shard-0.db -http-addr=127.0.0.2:8080 -config-file=sharding.toml -shard=shard-0 &
./distribKV -db-location=shard-0-r1.db -http-addr=127.0.0.22:8080 -config-file=sharding.toml -shard=shard-0 -replica &
./distribKV -db-location=shard-0-r2.db -http-addr=127.0.0.23:8080 -config-file=sharding.toml -shard=shard-0 -replica &

./distribKV -db-location=shard-1.db -http-addr=127.0.0.3:8080 -config-file=sharding.toml -shard=shard-1 &
./distribKV -db-location=shard-1-r1.db -http-addr=127.0.0.33:8080 -config-file=sharding.toml -shard=shard-1 -replica &

./distribKV -db-location=shard-2.db -http-addr=127.0.0.4:8080 -config-file=sharding.toml -shard=shard-2 &
./distribKV -db-location=shard-2-r1.db -http-addr=127.0.0.44:8080 -config-file=sharding.toml -shard=shard-2 -replica &
./distribKV -db-location=shard-2-r2.db -http-addr=127.0.0.45:8080 -config-file=sharding.toml -shard=shard-2 -replica &

./distribKV -db-location=shard-3.db -http-addr=127.0.0.5:8080 -config-file=sharding.toml -shard=shard-3 &
./distribKV -db-location=shard-3-r1.db -http-addr=127.0.0.55:8080 -config-file=sharding.toml -shard=shard-3 -replica &

wait
