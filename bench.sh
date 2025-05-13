go run cmd/bench/main.go \
  -shard-addrs="localhost:8080,localhost:8081,localhost:8082" \
  -iterations=5000 \
  -read-iterations=10000 \
  -concurrency=10
