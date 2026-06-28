# Custom Redis Server Clone

A high-performance, lightweight Redis server clone built from scratch to deeply understand low-level networking, protocols, and database storage engines. This project implements the core Redis RESP (REdis Serialization Protocol), in-memory data structures, replication, and data persistence mechanisms.

## 🚀 Features

### 1. Data Types
* **Strings**: Full support for key-value string pairs with fast lookups.
* **Lists**: Implemented linked-list structures allowing head/tail operations (`LPUSH`, `RPUSH`, `LPOP`, `RPOP`).

### 2. High Availability & Replication (Master/Replica)
* Implemented leader-follower **Replication architecture**.
* **Master Node**: Handles write commands and automatically propagates state changes to connected replicas.
* **Replica Node**: Operates in read-only mode, establishes handshakes with the master, and synchronizes data stream state asynchronously.

### 3. Persistence Engines
To ensure zero data loss, the server implements two standard Redis persistence models:
* **RDB (Redis Database File)**: Point-in-time snapshotting that serializes the in-memory dataset into a compact binary file at configured intervals.
* **AOF (Append Only File)**: Logs every write operation received by the server. Implements an append-only log that plays back transactions on server startup to reconstruct the exact original dataset.
