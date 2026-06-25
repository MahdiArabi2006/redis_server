package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"
)

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
			typ:   Array,
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

func handle_set(key string, value string, has_px bool, px int, connection net.Conn, isMaster bool, data []byte) {
	dbMu.Lock()

	DB[key] = value

	dbMu.Unlock()

	if has_px {
		delay := px * int(time.Millisecond)
		time.AfterFunc(time.Duration(delay), func() {
			dbMu.Lock()
			delete(DB, key)
			dbMu.Unlock()
		})
	}
	if isMaster {
		var buffer bytes.Buffer
		values := []string{"OK"}
		if write_RESP(SimpleString, values, false, 1, &buffer) == nil {
			connection.Write(buffer.Bytes())
		}
		propagateToReplicas(data)
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

func handle_replconf(connection net.Conn) {
	var buffer bytes.Buffer
	values := []string{"OK"}
	if write_RESP(SimpleString, values, false, 1, &buffer) == nil {
		connection.Write(buffer.Bytes())
	}
}

func handle_psync(connection net.Conn, config Config) {
	var emptyRDB, _ = hex.DecodeString("524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2")
	var buffer bytes.Buffer
	values := []string{"FULLRESYNC" + " " + config.masterReplid + " " + strconv.Itoa(config.masterReplOffset)}
	if write_RESP(SimpleString, values, false, 1, &buffer) == nil {
		connection.Write(buffer.Bytes())
	}
	connection.Write(append([]byte(fmt.Sprintf("$%d\r\n", len(emptyRDB))), emptyRDB...))
	addReplica(connection)
}

func handle_get_config(connection net.Conn, config Config, key string) {
	var buffer bytes.Buffer
	var values []string
	switch key {
	case "dir":
		values = []string{key, config.dir}
	case "dbfilename":
		values = []string{key, config.dbfilename}
	case "appendonly":
		values = []string{key, config.appendonly}
	case "appenddirname":
		values = []string{key, config.appenddirname}
	case "appendfilename":
		values = []string{key, config.appendfilename}
	case "appendfsync":
		values = []string{key, config.appendfsync}
	default:
		values = []string{key, config.dir}
	}
	if write_RESP(SimpleString, values, true, 2, &buffer) == nil {
		connection.Write(buffer.Bytes())
	}
}

func handle_key(connection net.Conn, regex string) {
	var buffer bytes.Buffer
	values := []string{}
	reg, error := regexp.Compile(regex)
	if error != nil {
		write_RESP(Error, []string{"ERR invalid regex"}, false, 0, &buffer)
		connection.Write(buffer.Bytes())
		return
	}

	dbMu.RLock()
	for key := range DB {
		if reg.MatchString(key) {
			values = append(values, key)
		}
	}
	dbMu.RUnlock()

	if write_RESP(SimpleString, values, true, len(values), &buffer) == nil {
		connection.Write(buffer.Bytes())
	}
}
