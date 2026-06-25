package main

import (
	"flag"
	"strconv"
	"strings"
	"os"
	"log"
	"fmt"
	"path/filepath"
	"io"
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

func loadFromAOF(config Config) error{
	aofMu.Lock()
	defer aofMu.Unlock()

	path, err := getActiveAOFFile(config.dir, config.appenddirname)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	aofFile = f

	reader := NewReader(aofFile)

	for {
		value, _, erro := reader.ReadValue()
		if erro == io.EOF{
			break
		}

		raw, err := encodeValue(value)
		if err != nil {
			continue
		}

		handleCommand(value, nil, config, raw,true)
	}
	return nil
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

	if config.appendonly == "yes" {
		path := filepath.Join(config.dir, config.appenddirname)

		err := os.MkdirAll(path, 0755)
		if err != nil {
			log.Fatalf("Failed to initialize directory for AOF: %v", err)
		}

		if config.appendfilename != "" {
			aofPath := filepath.Join(path, config.appendfilename+".1.incr.aof")
			
			if _, err := os.Stat(aofPath); os.IsNotExist(err) {
				file, err := os.Create(aofPath)
				if err != nil {
					log.Fatalf("Failed to create AOF file: %v", err)
				}
				file.Close()
				log.Println("New AOF file has been created")
			} else {
				log.Println("AOF file already exists, skipping creation")
			}

			manifPath := filepath.Join(path, "appendonly.manifest")
			
			if _, err := os.Stat(manifPath); os.IsNotExist(err) {
				manifFile, err := os.OpenFile(manifPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
				if err != nil {
					log.Printf("Error creating manifest file: %v", err)
					os.Exit(1)
				}
				content := fmt.Sprintf("file %s.1.incr.aof seq 1 type i\n", config.appendfilename)
				_, err = manifFile.WriteString(content)
				if err != nil {
					log.Printf("Error writing to manifest: %v", err)
				}
				manifFile.Close()
				log.Println("Manifest file created")
			} else {
				log.Println("Manifest file already exists, skipping")
				err = loadFromAOF(config)
				if err != nil{
					log.Println("Error initialize DB from AOF")
					os.Exit(1)
				}
			}
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
