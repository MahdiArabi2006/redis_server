package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var aofFile *os.File
var aofMu sync.Mutex

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

func writeResponse(type_string Type, response []string, is_array bool, size int, connection net.Conn) {
	var buffer bytes.Buffer
	if write_RESP(type_string, response, is_array, size, &buffer) == nil {
		connection.Write(buffer.Bytes())
	}
}

func getActiveAOFFile(dir string, appendDirName string) (string, error) {
	manifestPath := filepath.Join(dir, appendDirName, "appendonly.manifest")
	file, err := os.Open(manifestPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var activeFile string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "type i") {
			parts := strings.Split(line, " ")
			activeFile = parts[1]
		}
	}

	if activeFile == "" {
		return "", fmt.Errorf("no active incr aof file found")
	}
	return filepath.Join(dir, appendDirName, activeFile), nil
}

func write_to_aof(data []byte, config Config) error {
	aofMu.Lock()
	defer aofMu.Unlock()

	if aofFile == nil {
		path, err := getActiveAOFFile(config.dir, config.appenddirname)
		if err != nil {
			return err
		}

		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		aofFile = f
	}

	_, err := aofFile.Write(data)
	if err != nil {
		return err
	}

	if config.appendfsync == "always" {
		aofFile.Sync()
	}

	return nil
}

func handle_ping(connection net.Conn) {
	values := []string{"PONG"}
	writeResponse(SimpleString, values, false, 1, connection)
}

func handle_echo(connection net.Conn, value string) {
	values := []string{value}
	writeResponse(SimpleString, values, false, 1, connection)
}

func handle_set(key string, value string, has_px bool, px int, connection net.Conn, isMaster bool, data []byte, config Config, isAOFLoading bool) {
	DB.Lock()

	DB.Data[key] = &Entry{
		Type:  StringType,
		Value: value,
	}

	DB.Unlock()

	if has_px {
		delay := px * int(time.Millisecond)
		time.AfterFunc(time.Duration(delay), func() {
			DB.Lock()
			delete(DB.Data, key)
			DB.Unlock()
		})
	}
	if isMaster {
		if !isAOFLoading {
			write_to_aof(data, config)
			values := []string{"OK"}
			writeResponse(SimpleString, values, false, 1, connection)
		}
		propagateToReplicas(data)
	}
}

func handle_get(connection net.Conn, key string) {
	DB.RLock()
	entry := DB.Data[key]
	DB.RUnlock()

	if entry.Type != StringType {
		writeResponse(Error, []string{"it is not a string type"}, false, 0, connection)
	}

	values := []string{entry.Value.(string)}
	writeResponse(SimpleString, values, false, 1, connection)
}

func handle_prush(connection net.Conn, number_of_elements int, list_id string, array_values []Value) {
	DB.Lock()
	defer DB.Unlock()

	entry, ok := DB.Data[list_id]

	if !ok {
		entry = &Entry{
			Type:  ListType,
			Value: []string{},
		}
		DB.Data[list_id] = entry
	}

	if entry.Type != ListType {
		writeResponse(Error, []string{"it is not a list type"}, false, 0, connection)
	}

	list := entry.Value.([]string)

	for i := range number_of_elements {
		list = append(list, string(array_values[2+i].str))
	}

	entry.Value = list

	values := []string{strconv.Itoa(len(list))}
	writeResponse(Integer, values, false, 1, connection)
}

func handle_lrange(connection net.Conn, start string, stop string, list_id string) {
	DB.Lock()
	defer DB.Unlock()

	start_index, err := strconv.Atoi(start)
	if err != nil {
		writeResponse(Error, []string{"start index must be an integer"}, false, 0, connection)
		return
	}
	stop_index, err := strconv.Atoi(stop)
	if err != nil {
		writeResponse(Error, []string{"start index must be an integer"}, false, 0, connection)
		return
	}

	entry, ok := DB.Data[list_id]

	if !ok {
		writeResponse(SimpleString, []string{}, true, 0, connection)
		return
	}

	if entry.Type != ListType {
		writeResponse(Error, []string{"it is not a list type"}, false, 0, connection)
	}

	list := entry.Value.([]string)

	if start_index < 0 {
		start_index = max(len(list)+start_index, 0)
	}

	if stop_index < 0 {
		stop_index = max(len(list)+stop_index, 0)
	}

	if start_index >= len(list) {
		writeResponse(SimpleString, []string{}, true, 0, connection)
		return
	}

	if stop_index >= len(list) {
		stop_index = len(list) - 1
	}

	if start_index > stop_index {
		writeResponse(SimpleString, []string{}, true, 0, connection)
		return
	}

	values := []string{}

	for i := start_index; i <= stop_index; i++ {
		values = append(values, list[i])
	}

	writeResponse(SimpleString, values, true, stop_index-start_index+1, connection)
}

func handle_llen(connection net.Conn, list_id string) {
	DB.Lock()
	defer DB.Unlock()

	entry, ok := DB.Data[list_id]

	if !ok {
		writeResponse(Integer, []string{"0"}, false, 1, connection)
		return
	}

	list := entry.Value.([]string)

	values := []string{strconv.Itoa(len(list))}
	writeResponse(Integer, values, false, 1, connection)
}

func handle_lpop(connection net.Conn, list_id string, multiple_pop bool, n int) {
	DB.Lock()
	defer DB.Unlock()

	entry, ok := DB.Data[list_id]

	if !ok {
		writeResponse(BulkString, []string{""}, false, 1, connection)
		return
	}

	list := entry.Value.([]string)

	if multiple_pop {
		size := len(list)
		if n > size {
			writeResponse(SimpleString, list, true, size, connection)
			list = list[:0]
			entry.Value = list
			return
		} else {
			writeResponse(SimpleString, list[0:n], true, n, connection)
			list = list[n:]
			entry.Value = list
			return
		}
	} else if len(list) > 0 {
		writeResponse(SimpleString, []string{list[0]}, false, 1, connection)
		list = list[1:]
		entry.Value = list
		return
	}
	writeResponse(BulkString, []string{""}, false, 1, connection)
}

func handle_replconf(connection net.Conn) {
	values := []string{"OK"}
	writeResponse(SimpleString, values, false, 1, connection)
}

func handle_psync(connection net.Conn, config Config) {
	var emptyRDB, _ = hex.DecodeString("524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2")
	values := []string{"FULLRESYNC" + " " + config.masterReplid + " " + strconv.Itoa(config.masterReplOffset)}
	writeResponse(SimpleString, values, false, 1, connection)
	connection.Write(append([]byte(fmt.Sprintf("$%d\r\n", len(emptyRDB))), emptyRDB...))
	addReplica(connection)
}

func handle_get_config(connection net.Conn, config Config, key string) {
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
	writeResponse(SimpleString, values, true, 2, connection)
}

func handle_key(connection net.Conn, regex string) {
	values := []string{}
	reg, error := regexp.Compile(regex)
	if error != nil {
		writeResponse(Error, []string{"ERR invalid regex"}, false, 0, connection)
		return
	}

	DB.RLock()
	for key, entry := range DB.Data {
		if entry.Type == StringType && reg.MatchString(key) {
			values = append(values, key)
		}
	}
	DB.RUnlock()

	writeResponse(SimpleString, values, true, len(values), connection)
}

func handle_type(connection net.Conn, key string) {
	DB.RLock()
	defer DB.RUnlock()

	entry, ok := DB.Data[key]

	if !ok {
		writeResponse(SimpleString, []string{"none"}, false, 1, connection)
		return
	}

	switch entry.Type {
	case 0:
		writeResponse(SimpleString, []string{"string"}, false, 0, connection)
	case 1:
		writeResponse(SimpleString, []string{"list"}, false, 0, connection)
	case 2:
		writeResponse(SimpleString, []string{"set"}, false, 0, connection)
	case 3:
		writeResponse(SimpleString, []string{"hash"}, false, 0, connection)
	case 4:
		writeResponse(SimpleString, []string{"zset"}, false, 0, connection)
	case 5:
		writeResponse(SimpleString, []string{"stream"}, false, 0, connection)
	case 6:
		writeResponse(SimpleString, []string{"vectorset"}, false, 0, connection)
	}
}
