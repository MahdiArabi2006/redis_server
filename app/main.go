package main

import (
	"fmt"
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

	fmt.Println("Logs from your program will appear here!")

	if config.IsReplica(){
		error := StartReplicationHandshake(config)
		if error != nil{
			fmt.Fprintf(os.Stderr, "handshake failed: %v\n", error)
			os.Exit(1)
		}
	}

	StartServer(config)
}
