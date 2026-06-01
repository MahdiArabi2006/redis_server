package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var (
	_ = net.Listen
	_ = os.Exit
)

func RESP_parser(buffer []byte, size int) (Value,error) {

	rd := NewReader(bytes.NewReader(buffer))

	value, _, err := rd.ReadValue()

	if err != nil {
		return value,err
	}

	return value,nil
}

func handleCommand(value Value, connection net.Conn) {
	switch value.typ{		
	case Array:
		if strings.ToLower(string(value.array[0].str)) == "echo"{
			connection.Write([]byte("+" + string( value.array[1].str) + "\r\n"))
		}
		if strings.ToLower(string(value.array[0].str)) == "ping"{
			connection.Write([]byte("+PONG\r\n"))
		}
	}
}

func handleClient(connection net.Conn) {
	defer connection.Close()

	buffer := make([]byte, 1024)

	for {
		size, error := connection.Read(buffer)
		if error != nil {
			break
		}

		value,err := RESP_parser(buffer, size)
		if err != nil{
			connection.Write([]byte("-ERR unknown protocol or bad syntax\r\n"))
			continue
		}

		handleCommand(value, connection)
	}
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment the code below to pass the first stage
	//
	listener, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	defer listener.Close()

	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleClient((connection))
	}
}
