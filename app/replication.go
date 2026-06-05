package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

func send_ping(connection net.Conn) {
	var buffer bytes.Buffer
	values := []string{"PING"}
	error := write_RESP(SimpleString, values, true, 1, &buffer)
	if error == nil {
		connection.Write(buffer.Bytes())
	}
}

func send_replconf_port(connection net.Conn, config Config) {
	var buffer bytes.Buffer
	values := []string{"REPLCONF", "listening-port", strconv.Itoa(config.Port)}
	error := write_RESP(SimpleString, values, true, 3, &buffer)
	if error == nil {
		connection.Write(buffer.Bytes())
	}
}

func send_replconf_capa(connection net.Conn) {
	var buffer bytes.Buffer
	values := []string{"REPLCONF", "capa", "psync2"}
	error := write_RESP(SimpleString, values, true, 3, &buffer)
	if error == nil {
		connection.Write(buffer.Bytes())
	}
}

func send_psync_command(connection net.Conn) {
	var buffer bytes.Buffer
	values := []string{"PSYNC", "?", "-1"}
	error := write_RESP(SimpleString, values, true, 3, &buffer)
	if error == nil {
		connection.Write(buffer.Bytes())
	}
}

func readCommandsFromMaster(connection net.Conn, config Config) {
	buffer := make([]byte, 1024)

	for {
		size, err := connection.Read(buffer)
		if err != nil {
			fmt.Println("master connection closed:", err)
			return
		}

		raw := make([]byte, size)
		copy(raw, buffer[:size])

		value, parseErr := RESP_parser(raw, len(raw))
		if parseErr != nil {
			fmt.Println("failed to parse command from master:", parseErr)
			continue
		}

		handleCommand(value, connection, config, raw)
	}
}


func StartReplicationHandshake(config Config) error {
	connenction, err := net.Dial("tcp", config.MasterHost+":"+config.MasterPort)
	if err != nil {
		return err
	}

	send_ping(connenction)
	buffer := make([]byte, 1024)
	size, error := connenction.Read(buffer)
	if error != nil {
		return error
	}

	value, err := RESP_parser(buffer, size)
	if err != nil {
		return err
	}

	if strings.ToLower(string(value.str)) != PONG {
		return errors.New("error handshaking while recivie PONG")
	}

	//fmt.Println(string(value.str))

	send_replconf_port(connenction, config)

	size, error = connenction.Read(buffer)
	if error != nil {
		return error
	}

	value, err = RESP_parser(buffer, size)
	if err != nil {
		return err
	}

	if strings.ToLower(string(value.str)) != OK {
		return errors.New("error handshaking while REPLCONF")
	}

	//fmt.Println(string(value.str))

	send_replconf_capa(connenction)

	size, error = connenction.Read(buffer)
	if error != nil {
		return error
	}

	value, err = RESP_parser(buffer, size)
	if err != nil {
		return err
	}

	if strings.ToLower(string(value.str)) != OK {
		return errors.New("error handshaking while REPLCONF")
	}

	//fmt.Println(string(value.str))

	send_psync_command(connenction)

	size, error = connenction.Read(buffer)
	if error != nil {
		return error
	}

	value, err = RESP_parser(buffer, size)
	if err != nil {
		return error
	}

	if strings.ToLower(strings.Split(string(value.str), " ")[0]) != FULLRESYNC {
		return errors.New("error handshaking while PSYNC")
	}

	//fmt.Println(string(value.str))

	size, error = connenction.Read(buffer)
	if error != nil {
		return error
	}

	go readCommandsFromMaster(connenction, config)

	return nil
}
