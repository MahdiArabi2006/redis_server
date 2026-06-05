package main

import(
	"strings"
	"strconv"
	"net"
)

const (
	PING string = "ping"
	ECHO string = "echo"
	SET string = "set"
	GET string = "get"
	PRUSH string = "prush"
)

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