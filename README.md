
# A Distributed Key-Value Store in Go

<div align="center">
  <img src="img/replica.png" alt="distribKV Architecture" width="800">
  <p><i>distribKV Architecture: the sharding and replication model</i></p>
</div>

**distribKV** is a distributed key-value store written in Go. It supports sharding, replication, client-side routing, LSM-based storage (BadgerDB), and more.


---

> [!NOTE]
> This project is for **educational purposes** — to understand how a **key-value store** works under the hood along with the theoretical concepts of **sharding, replication, LSM trees, consistency models**, **bloom filters**, and **distributed systems**.


## 📦 Features

- **Static Sharding** - Horizontally partition data across multiple nodes to distribute load
- **Leader-Follower Replication** - Fault tolerance with automatic leader-based replication
- **Client-Side Routing** - Hash-based routing to efficiently direct requests to correct shards
- **LSM-Tree Storage Engine** - High-performance persistence using BadgerDB with optimized read/write paths
- **Benchmarking Tools** - Comprehensive suite to measure throughput, latency, and scaling characteristics
- **Concurrent Request Handling** - Efficiently process parallel operations with Go's lightweight goroutines
- **HTTP API** - Simple REST-style interface for key-value operations
- **Configuration-Driven** - TOML-based configuration for easy shard setup and management
- **Persistence** with Write-Ahead Logging and Compaction
- Built with Go 1.24 and designed for extensibility

---

> [!NOTE]
> This project is primarily for **educational purposes** to demonstrate distributed systems concepts including sharding, replication, LSM trees, bloom filters, and consistency models.

---

## Architecture

distribKV follows a multi-layered architecture designed for scalability and resilience:

### High-Level Components:

1. **Client Layer**: Handles request routing based on key hashing
2. **API Layer**: HTTP endpoints for key-value operations
3. **Sharding Layer**: Distributes data across nodes using consistent hashing
4. **Replication Layer**: Ensures data redundancy with leader-follower model
5. **Storage Layer**: BadgerDB-backed persistent storage with LSM-tree implementation

### Data Flow:

1. Client sends a request (GET/SET) to any node in the cluster
2. The receiving node determines the appropriate shard using the key hash
3. If the current node is not responsible for the key, it redirects the request
4. For write operations:
   - The leader node processes the write request
   - The change is persisted to the local BadgerDB instance
   - A replication entry is created for followers to pick up
5. For read operations:
   - Both leader and replica nodes can serve read requests
   - Reads on replicas may return slightly stale data (eventual consistency)

--- 

## 💻 System Design

### Sharding

distribKV implements static sharding where the mapping between keys and shards is defined in a configuration file (`sharding.toml`). Each key is deterministically assigned to a shard using a consistent hashing algorithm.

```toml
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
```

**Sharding Implementation:**
- Key hashing uses a consistent algorithm to ensure even distribution
- Each shard is responsible for a specific range of the key space
- Shard configuration is loaded at startup and remains static during runtime
- Future versions may support dynamic shard rebalancing

### Replication

Replication follows a primary-secondary (leader-follower) model:

- **Leader Node**: Handles all write operations for its shard
- **Follower Nodes**: Asynchronously replicate data from the leader
- **Eventual Consistency**: Changes propagate to followers over time

**Replication Process:**
1. When a leader receives a write operation, it:
   - Persists the change locally
   - Adds the change to a replication queue
2. Follower nodes periodically poll their leader for new changes
3. When followers receive changes, they apply them to their local store
4. Followers acknowledge successful replication to the leader
5. Leaders can prune acknowledged entries from the replication queue

### Storage (BadgerDB & LSM Trees)

distribKV leverages [BadgerDB](https://github.com/dgraph-io/badger) as its storage engine, which implements a Log-Structured Merge (LSM) Tree:

**Components:**
- **MemTable**: In-memory sorted table for recent writes
- **Sorted String Tables (SSTables)**: Immutable disk files for persisted data
- **Write-Ahead Log (WAL)**: Transaction log ensuring durability
- **Bloom Filters**: Probabilistic data structure to optimize read performance

**Write Path:**
1. Incoming writes are first logged to the WAL
2. Data is inserted into the in-memory MemTable
3. When the MemTable reaches capacity, it's flushed to disk as an immutable SSTable
4. Metadata is updated to track the new SSTable

**Read Path:**
1. First check the MemTable for recent writes
2. If not found, check SSTables from newest to oldest
3. Bloom filters quickly eliminate SSTables that don't contain the key
4. Return the value when found or null if not exists

**Compaction Process:**
- Background process periodically merges multiple SSTables
- Removes deleted entries and consolidates updates
- Improves read performance by reducing the number of files to check
- Reclaims disk space by removing obsolete data

### Client Routing

distribKV implements client-side routing to direct requests to the appropriate shard:

1. Client hashes the key to determine the target shard
2. If the request reaches a non-target node, it's redirected with HTTP 307/308
3. Redirection includes the target node's address for direct future access
4. Clients can optionally cache shard mapping for more efficient routing

---

## API Endpoints

The API is HTTP-based with the following endpoints:

| Endpoint | Method | Parameters | Description |
|----------|--------|------------|-------------|
| `/get` | GET | `key` (required) | Retrieve a value for the given key |
| `/set` | GET | `key` (required), `value` (required) | Store a key-value pair |
| `/purge` | GET | None | Remove keys that don't belong to the current shard |
| `/next-replication-key` | GET | None | (Internal) Used by replicas to fetch updates |
| `/delete-replication-key` | GET | `key` (required), `value` (required) | (Internal) Acknowledge successful replication |
| `/healthz` | GET | None | Health check endpoint |

**Example Usage:**

```bash
# Set a key-value pair
curl "http://127.0.0.2:8080/set?key=user123&value=John%20Doe"

# Get a value
curl "http://127.0.0.2:8080/get?key=user123"

# Health check
curl "http://127.0.0.2:8080/healthz"
```
---

## Consistency Model

distribKV implements **eventual consistency** to prioritize availability and partition tolerance:

### Consistency Characteristics:

- **Write Path**: All writes go to the leader node first
- **Replication**: Changes are asynchronously propagated to replicas
- **Read Freshness**: Reads from replicas may return stale data temporarily
- **Convergence**: All replicas eventually reach the same state
- **No Ordering Guarantees**: The system doesn't enforce global ordering of operations

### Concurrency & Benchmarking

- **Goroutines**  
  Efficient parallelism using lightweight Go goroutines for concurrent `GET`/`SET` requests.

- **Benchmarking**  
  Tools for measuring:
  - Throughput (requests/sec)
  - Latency (response times)
  - Load performance

---

### Fault Tolerance & Resilience

- **Replication**  
  Replicas allow the system to survive leader failures without data loss.

- **Data Synchronization**  
  Periodic syncing ensures eventual consistency between leader and replicas.

---

### CAP Theorem Trade-offs

> “You can only choose two: **Consistency**, **Availability**, **Partition Tolerance**.”

**distribKV** favors:

- ✅ **Availability**
- ✅ **Partition Tolerance**
- 🚫 **(Relaxed) Consistency**

Strong consistency can be added with protocols like Raft or Paxos in the future (LOL).

---

## References

This project draws inspiration from:
- [Go, for Distributed Systems by Russ Cox](https://go.dev/talks/2013/distsys.slide#1)
- [Designing Data-Intensive Applications by Martin Kleppmann](https://www.amazon.com/Designing-Data-Intensive-Applications-Reliable-Maintainable/dp/1449373321)
- [Patterns of Distributed Systems by Unmesh Joshi](https://martinfowler.com/books/patterns-distributed.html)
- [Distributed key-value database series on YouTube](https://www.youtube.com/playlist?list=PLWwSgbaBp9XrMkjEhmTIC37WX2JfwZp7I)
- [BadgerDB docs](https://docs.hypermode.com/badger/overview)
- [Go package for bloom filters](https://github.com/bits-and-blooms/bloom)
- [Arpit Bhayani short tutorials](https://www.youtube.com/@AsliEngineering)
- [Consistent Hashing](https://www.hellointerview.com/learn/system-design/deep-dives/consistent-hashing)


