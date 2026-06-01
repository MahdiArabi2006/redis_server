package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var (
	_ = net.Listen
	_ = os.Exit
)

func RESP_parser(buffer []byte, size int) (Value, error) {

	rd := NewReader(bytes.NewReader(buffer))

	value, _, err := rd.ReadValue()

	if err != nil {
		return value, err
	}

	return value, nil
}

func handle_set(key string, value string, px *int, db map[string]string) {
	db[key] = value

	if px != nil {
		delay := (*px) * int(time.Millisecond)
		time.AfterFunc(time.Duration(delay), func() {
			delete(db, key)
		})
	}
}

func handleCommand(value Value, connection net.Conn, db map[string]string, lists map[string][]string) {
	switch value.typ {
	case Array:
		if strings.ToLower(string(value.array[0].str)) == "echo" {
			connection.Write([]byte("+" + string(value.array[1].str) + "\r\n"))
		}
		if strings.ToLower(string(value.array[0].str)) == "ping" {
			connection.Write([]byte("+PONG\r\n"))
		}
		if strings.ToLower(string(value.array[0].str)) == "set" {
			if len(value.array) > 3 {
				if strings.ToLower(string(value.array[3].str)) == "px" {
					px, err := strconv.Atoi(string(value.array[4].str))
					if err != nil {
						connection.Write([]byte("-px must be an positive integer\r\n"))
						return
					}
					handle_set((string(value.array[1].str)), string(value.array[2].str), &px, db)
				}
			} else {
				db[string(value.array[1].str)] = string(value.array[2].str)
			}
			connection.Write([]byte("+OK\r\n"))
		}
		if strings.ToLower(string(value.array[0].str)) == "get" {
			connection.Write([]byte("+" + db[string(value.array[1].str)] + "\r\n"))
		}
		if strings.ToLower(string(value.array[0].str)) == "prush" {
			number_of_elements := len(value.array) - 2
			for i := range number_of_elements{
				lists[string(value.array[1].str)] = append(lists[string(value.array[1].str)], string(value.array[2 + i].str))	
			}
			connection.Write([]byte(":" + strconv.Itoa(len(lists[string(value.array[1].str)])) + "\r\n"))
		}
	}
}

func handleClient(connection net.Conn) {
	defer connection.Close()

	buffer := make([]byte, 1024)

	db := make(map[string]string)
	lists := map[string][]string{}

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

		handleCommand(value, connection, db, lists)
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
