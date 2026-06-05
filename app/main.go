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

var DB = make(map[string]string)
var dbMu sync.RWMutex
var Lists = map[string][]string{}
var listsMu sync.RWMutex

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
