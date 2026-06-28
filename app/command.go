package main

import (
	"net"
	"strconv"
	"strings"
)

const (
	PING       string = "ping"
	PONG       string = "pong"
	ECHO       string = "echo"
	SET        string = "set"
	GET        string = "get"
	PRUSH      string = "prush"
	LRANGE     string = "lrange"
	LPUSH      string = "lpush"
	LLEN       string = "llen"
	LPOP       string = "lpop"
	REPLCONF   string = "replconf"
	OK         string = "ok"
	PSYNC      string = "psync"
	FULLRESYNC string = "fullresync"
	CONFIG     string = "config"
	KEY        string = "keys"
)

func handleCommand(value Value, connection net.Conn, config Config, raw_binary []byte, isAOFLoading bool) {
	switch value.typ {
	case Array:
		command := strings.ToLower(string(value.array[0].str))

		if command == ECHO {
			handle_echo(connection, string(value.array[1].str))
		}
		if command == PING {
			handle_ping(connection)
		}
		if command == SET {
			if len(value.array) > 3 {
				if strings.ToLower(string(value.array[3].str)) == "px" {
					px, err := strconv.Atoi(string(value.array[4].str))
					if err != nil {
						connection.Write([]byte("-px must be an positive integer\r\n"))
						return
					}
					handle_set((string(value.array[1].str)), string(value.array[2].str), true, px, connection, !config.IsReplica(), raw_binary, config, isAOFLoading)
				}
			} else {
				handle_set((string(value.array[1].str)), string(value.array[2].str), false, 0, connection, !config.IsReplica(), raw_binary, config, isAOFLoading)
			}
		}
		if command == GET {
			handle_get(connection, string(value.array[1].str))
		}
		if command == PRUSH {
			handle_prush(connection, len(value.array)-2, string(value.array[1].str), value.array)
		}
		if command == LRANGE {
			handle_lrange(connection, string(value.array[2].str), string(value.array[3].str), string(value.array[1].str))
		}
		if command == LPUSH {

		}
		if command == LLEN {
			handle_llen(connection, string(value.array[1].str))
		}
		if command == LPOP {
			if len(value.array) > 2 {
				n, err := strconv.Atoi(string(value.array[2].str))
				if err != nil {
					connection.Write([]byte("pass a number\r\n"))
					return
				}
				handle_lpop(connection, string(value.array[1].str), true, n)
			} else {
				handle_lpop(connection, string(value.array[1].str), false, 0)
			}
		}
		if command == REPLCONF {
			handle_replconf(connection)
		}
		if command == PSYNC {
			handle_psync(connection, config)
		}
		if command == CONFIG {
			if string(value.array[1].str) == "GET" {
				handle_get_config(connection, config, string(value.array[2].str))
			}
		}
		if command == KEY {
			handle_key(connection, string(value.array[1].str))
		}
	}
}
