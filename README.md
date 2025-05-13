# distribKV

distribKV is a distributed key-value database implemented in Go. It supports horizontal scaling through sharding and ensures high availability via leader-based replication. The system is designed for resilience, scalability, and performance, making it suitable for learning and experimentation with distributed systems concepts.

[![Directory docs](img/replica.png)](https://github.com/Sagor0078/distribKV)



## Core Theoretical Concepts
1. Distributed Systems Fundamentals

    Sharding: Data is partitioned across multiple nodes based on a hash or shard index, reducing load and improving scalability.

    Replication: Maintains copies (replicas) of data for fault tolerance and high availability. Typically employs a leader–follower model where the leader handles writes, and replicas synchronize data.

2. Consistency Models

    Eventual Consistency: After a write, data is eventually replicated and becomes consistent across all nodes.

    Leader-based Replication: A designated node accepts writes, and followers replicate data from it, ensuring consistency.

3. Client-side Routing & Redirection

    Clients hash keys to determine the appropriate shard to contact.

    If a request reaches the wrong shard, it may be redirected using HTTP status codes like 307 or 308.

4. Persistence and Storage

    LSM Tree Concepts via BadgerDB: Utilizes a write-optimized storage engine inspired by RocksDB/LevelDB for efficient data storage.

    Write-Ahead Logging & Compaction: Ensures durability and facilitates crash recovery through logging and data compaction mechanisms.

5. Concurrency and Benchmarking

    Handles concurrent GET/SET operations using Goroutines for efficient parallel processing.

    Includes benchmarking tools to measure throughput, latency, and performance under various workloads.

6. Fault Tolerance & Resilience

    Employs redundancy via replicas to maintain operation continuity even if a leader node fails.

    Implements data synchronization strategies to keep replicas up to date and consistent.

7. CAP Theorem

    Demonstrates trade-offs among Consistency, Availability, and Partition Tolerance.

    This system favors availability and partition tolerance, while sacrificing strict consistency unless strong synchronization protocols are added.