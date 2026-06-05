package main
import(
	"sync"
	"net"
)

var replicasMu sync.RWMutex
var replicas = make(map[net.Conn]struct{})

func addReplica(conn net.Conn) {
	replicasMu.Lock()
	defer replicasMu.Unlock()

	replicas[conn] = struct{}{}
}

func removeReplica(conn net.Conn) {
	replicasMu.Lock()
	defer replicasMu.Unlock()

	delete(replicas, conn)
}

func propagateToReplicas(data []byte) {
	replicasMu.RLock()
	snapshot := make([]net.Conn, 0, len(replicas))
	for conn := range replicas {
		snapshot = append(snapshot, conn)
	}
	replicasMu.RUnlock()

	for _, conn := range snapshot {
		if _, err := conn.Write(data); err != nil {
			removeReplica(conn)
			conn.Close()
		}
	}
}
