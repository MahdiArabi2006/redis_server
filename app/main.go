package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
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

func RESP_parser(buffer []byte, size int) (Value, error) {

	rd := NewReader(bytes.NewReader(buffer))

	value, _, err := rd.ReadValue()

	if err != nil {
		return value, err
	}

	return value, nil
}

func write_RESP(type_value Type, strings []string, is_array bool, size int, buffer *bytes.Buffer) error {
	wr := NewWriter(buffer)
	if is_array {
		array := []Value{}
		for i := range size {
			val := Value{
				typ: type_value,
				str: []byte(strings[i]),
			}
			array = append(array, val)
		}
		value := Value{
			typ:   type_value,
			array: array,
		}
		error := wr.WriteValue(value)
		return error

	} else {
		if type_value == Integer {
			number, err := strconv.Atoi(strings[0])
			if err != nil {
				return err
			}
			value := Value{
				typ:     type_value,
				integer: number,
			}
			error := wr.WriteValue(value)
			return error
		} else {
			value := Value{
				typ: type_value,
				str: []byte(strings[0]),
			}
			error := wr.WriteValue(value)
			return error
		}
	}
}

func handle_ping(connection net.Conn) {
	var buffer bytes.Buffer
	values := []string{"PONG"}
	if write_RESP(SimpleString, values, false, 1, &buffer) == nil {
		connection.Write(buffer.Bytes())
	}
}

func handle_echo(connection net.Conn, value string) {
	var buffer bytes.Buffer
	values := []string{value}
	if write_RESP(SimpleString, values, false, 1, &buffer) == nil {
		connection.Write(buffer.Bytes())
	}
}

func handle_set(key string, value string, has_px bool, px int, connection net.Conn) {
	dbMu.Lock()
	defer dbMu.Unlock()

	DB[key] = value

	if has_px {
		delay := px * int(time.Millisecond)
		time.AfterFunc(time.Duration(delay), func() {
			delete(DB, key)
		})
	}

	var buffer bytes.Buffer
	values := []string{"OK"}
	if write_RESP(SimpleString, values, false, 1, &buffer) == nil {
		connection.Write(buffer.Bytes())
	}
}

func handle_get(connection net.Conn, key string) {
	dbMu.RLock()
	defer dbMu.RUnlock()

	var buffer bytes.Buffer
	values := []string{DB[key]}
	if write_RESP(SimpleString, values, false, 1, &buffer) == nil {
		connection.Write(buffer.Bytes())
	}
}

func handle_prush(connection net.Conn, number_of_elements int, list_id string, array_values []Value) {
	listsMu.Lock()
	defer listsMu.Unlock()

	for i := range number_of_elements {
		Lists[list_id] = append(Lists[list_id], string(array_values[2+i].str))
	}

	var buffer bytes.Buffer
	values := []string{strconv.Itoa(len(Lists[list_id]))}
	if write_RESP(Integer, values, false, 1, &buffer) == nil {
		connection.Write(buffer.Bytes())
	}
}

func handleCommand(value Value, connection net.Conn) {
	switch value.typ {
	case Array:
		if strings.ToLower(string(value.array[0].str)) == "echo" {
			handle_echo(connection, string(value.array[1].str))
		}
		if strings.ToLower(string(value.array[0].str)) == PING {
			handle_ping(connection)
		}
		if strings.ToLower(string(value.array[0].str)) == SET {
			if len(value.array) > 3 {
				if strings.ToLower(string(value.array[3].str)) == "px" {
					px, err := strconv.Atoi(string(value.array[4].str))
					if err != nil {
						connection.Write([]byte("-px must be an positive integer\r\n"))
						return
					}
					handle_set((string(value.array[1].str)), string(value.array[2].str), true, px, connection)
				}
			} else {
				handle_set((string(value.array[1].str)), string(value.array[2].str), false, 0, connection)
			}
		}
		if strings.ToLower(string(value.array[0].str)) == GET {
			handle_get(connection, string(value.array[1].str))
		}
		if strings.ToLower(string(value.array[0].str)) == PRUSH {
			handle_prush(connection, len(value.array)-2, string(value.array[1].str), value.array)
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

		value, err := RESP_parser(buffer, size)
		if err != nil {
			connection.Write([]byte("-ERR unknown protocol or bad syntax\r\n"))
			continue
		}

		handleCommand(value, connection)
	}
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	port := flag.Int("port",6379,"TCP Port")
	flag.Parse()
	fmt.Println("Logs from your program will appear here!")

	// Uncomment the code below to pass the first stage
	//
	listener, err := net.Listen("tcp", "0.0.0.0:" + strconv.Itoa((*port)))
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
