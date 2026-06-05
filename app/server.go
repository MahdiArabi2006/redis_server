package main

import (
	"fmt"
	"net"
	"os"
)

func handleClient(connection net.Conn, config Config) {
	defer connection.Close()

	reader := NewReader(connection)

	for {
		value, _, erro := reader.ReadValue()
		if erro != nil {
			break
		}
		raw, err := encodeValue(value)
		if err != nil {
			connection.Write([]byte("-ERR failed to encode command\r\n"))
			continue
		}

		handleCommand(value, connection, config, raw)
	}
}

func StartServer(config Config) {
	listener, err := net.Listen("tcp", config.Address())
	if err != nil {
		fmt.Println("Failed to bind to port", config.Port)
		os.Exit(1)
	}
	defer listener.Close()

	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			continue
		}

		go handleClient(connection,config)
	}
}
