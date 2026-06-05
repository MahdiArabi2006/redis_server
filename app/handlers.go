package main

import (
	"bytes"
	"net"
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

func handle_replconf(connection net.Conn) {
	var buffer bytes.Buffer
	values := []string{"OK"}
	if write_RESP(SimpleString, values, false, 1, &buffer) == nil {
		connection.Write(buffer.Bytes())
	}
}

func handle_psync(connection net.Conn, config Config){
	var buffer bytes.Buffer
	values := []string{"FULLRESYNC" + " " + config.masterReplid + " " + strconv.Itoa(config.masterReplOffset)}
	if write_RESP(SimpleString, values, false, 1, &buffer) == nil {
		connection.Write(buffer.Bytes())
	}
}