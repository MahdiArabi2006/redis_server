# Custom Redis Server Clone

A high-performance, lightweight Redis server clone built from scratch to deeply understand low-level networking, protocols, and database storage engines. This project implements the core Redis RESP (REdis Serialization Protocol), in-memory data structures, replication, and data persistence mechanisms.

## Features

### 1. Data Types
The codebase is designed with a modular structure, meaning **additional data types can be easily integrated** by implementing the core storage interfaces. The currently implemented types serve as a robust sample/proof-of-concept:
* **Strings**: Full support for key-value string pairs with fast lookups.
* **Lists**: Implemented linked-list structures allowing head/tail operations.

### 2. High Availability & Replication (Master/Replica)
* Implemented leader-follower **Replication architecture**.
* **Master Node**: Handles write commands and automatically propagates state changes to connected replicas.
* **Replica Node**: Operates in read-only mode, establishes handshakes with the master, and synchronizes data stream state asynchronously.

### 3. Persistence Engines
To ensure zero data loss, the server implements two standard Redis persistence models:
* **RDB (Redis Database File)**: Point-in-time snapshotting that serializes the in-memory dataset into a compact binary file at configured intervals.
* **AOF (Append Only File)**: Logs every write operation received by the server. Implements an append-only log that plays back transactions on server startup to reconstruct the exact original dataset.

---

## Tech Stack & Architecture

* **Language:** Go (Golang)
* **Networking:** Low-level TCP Socket programming utilizing Go's efficient, concurrent `net` package and goroutines.
* **Protocol:** Custom RESP parser supporting Bulk Strings, Arrays, and Integers.

---

## Getting Started

### Prerequisites
* Go 1.20 or higher installed.

### Supported Configuration Flags
The executable parses standard Redis command-line flags on startup:
* `--port` : The TCP port on which the Redis server will listen (Default: `6379`).
* `--replicaof` : Master server address formatted as `"master_host master_port"` (e.g., `"127.0.0.1 6379"`).
* `--dir` : The path to the directory where the RDB and AOF files are stored.
* `--dbfilename` : The name of the RDB snapshot file.
* `--appendonly` : Controls whether AOF persistence is enabled (`yes` or `no`, Default: `no`).
* `--appenddirname` : The subdirectory under `dir` where AOF and manifest files are stored (Default: `appendonlydir`).
* `--appendfilename` : The name of the append-only file that records write operations (Default: `appendonly.aof`).
* `--appendfsync` : How often buffered writes are flushed to the AOF file on disk (`always`, `everysec`, `no`, Default: `everysec`).

---
