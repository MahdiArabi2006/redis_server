package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var (
	_ = net.Listen
	_ = os.Exit
)

type ValueType int

const (
    StringType    ValueType = 0
    ListType      ValueType = 1
    SetType       ValueType = 2
    HashType      ValueType = 3
    ZSetType      ValueType = 4
    StreamType    ValueType = 5
    VectorSetType ValueType = 6
)

type Entry struct {
    Type  ValueType
    Value any
}

type Store struct {
    sync.RWMutex
    Data map[string]*Entry
}

var DB = Store{
    Data: make(map[string]*Entry),
}

func main() {
	config := LoadConfig()

	if config.dir != "" && config.dbfilename != "" {
		err := loadRDBIntoStore(config)
		if err != nil {
			log.Println("RDB load failed:", err)
		} else {
			log.Println("RDB loaded successfully")
		}
	}

	fmt.Println("Logs from your program will appear here!")

	if config.IsReplica() {
		error := StartReplicationHandshake(config)
		if error != nil {
			fmt.Fprintf(os.Stderr, "handshake failed: %v\n", error)
			os.Exit(1)
		}
	}

	StartServer(config)
}
