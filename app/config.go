package main

import (
	"flag"
	"strconv"
	"strings"
)

type Config struct {
	Port             int
	ReplicaOf        string
	MasterHost       string
	MasterPort       string
	masterReplid     string
	masterReplOffset int
	dir              string
	dbfilename       string
	appendonly       string
	appenddirname    string
	appendfilename   string
	appendfsync      string
}

func LoadConfig() Config {
	port := flag.Int("port", 6379, "TCP Port")
	replicaof := flag.String("replicaof", "", "Master server address")
	dir := flag.String("dir", "", "the path to the directory where the RDB file is stored")
	dbfilename := flag.String("dbfilename", "", "the name of the RDB file ")
	appendonly := flag.String("appendonly", "no", "Controls whether AOF persistence is enabled or disabled")
	appenddirname := flag.String("appenddirname", "appendonlydir", "The subdirectory under dir where AOF and manifest files are stored")
	appendfilename := flag.String("appendfilename", "appendonly.aof", "The name of the append-only file that records write operations")
	appendfsync := flag.CommandLine.String("appendfsync", "everysec", "How often buffered writes are flushed to the AOF file on disk")
	flag.Parse()

	config := Config{
		Port:           *port,
		ReplicaOf:      *replicaof,
		dir:            *dir,
		dbfilename:     *dbfilename,
		appendonly:     *appendonly,
		appenddirname:  *appenddirname,
		appendfilename: *appendfilename,
		appendfsync:    *appendfsync,
	}

	if config.ReplicaOf != "" {
		parts := strings.Fields(config.ReplicaOf)
		if len(parts) == 2 {
			config.MasterHost = parts[0]
			config.MasterPort = parts[1]
		}
	} else {
		config.masterReplid = "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb"
		config.masterReplOffset = 0
	}

	return config
}

func (config Config) Address() string {
	return "0.0.0.0:" + strconv.Itoa(config.Port)
}

func (config Config) IsReplica() bool {
	return config.ReplicaOf != ""
}
