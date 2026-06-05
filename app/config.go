package main

import (
	"flag"
	"strconv"
	"strings"
)

type Config struct {
	Port      int
	ReplicaOf string
	MasterHost string
	MasterPort string
}

func LoadConfig() Config {
	port := flag.Int("port", 6379, "TCP Port")
	replicaof := flag.String("replicaof", "", "Master server address")
	flag.Parse()

	config := Config{
		Port:      *port,
		ReplicaOf: *replicaof,
	}

	if config.ReplicaOf != "" {
		parts := strings.Fields(config.ReplicaOf)
		if len(parts) == 2 {
			config.MasterHost = parts[0]
			config.MasterPort = parts[1]
		}
	}

	return config
}

func (config Config) Address() string {
	return "0.0.0.0:" + strconv.Itoa(config.Port)
}

func (config Config) IsReplica() bool {
	return config.ReplicaOf != ""
}
