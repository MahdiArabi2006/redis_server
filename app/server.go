package main

import (
	"fmt"
	"net"
	"os"
)

func handleClient(connection net.Conn, config Config) {
	defer connection.Close()

	buffer := make([]byte, 1024)

	for {
		size, error := connection.Read(buffer)
		if error != nil {
			break
		}

		value, err := RESP_parser(buffer, size)
		if err != nil {
			connection.Write([]byte("-ERR unknown protocol or bad syntax\r\n"))
			continue
		}

		handleCommand(value, connection,config)
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
