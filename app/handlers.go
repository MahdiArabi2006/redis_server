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
		if !isAOFLoading {
			write_to_aof(data, config)
			values := []string{"OK"}
			writeResponse(SimpleString, values, false, 1, connection)
		}
		propagateToReplicas(data)
	}
}

func handle_get(connection net.Conn, key string) {
	dbMu.RLock()
	defer dbMu.RUnlock()

	values := []string{DB[key]}
	writeResponse(SimpleString, values, false, 1, connection)
}

func handle_prush(connection net.Conn, number_of_elements int, list_id string, array_values []Value) {
	listsMu.Lock()
	defer listsMu.Unlock()

	for i := range number_of_elements {
		Lists[list_id] = append(Lists[list_id], string(array_values[2+i].str))
	}

	values := []string{strconv.Itoa(len(Lists[list_id]))}
	writeResponse(Integer, values, false, 1, connection)
}

func handle_lrange(connection net.Conn, start string, stop string, list_id string) {
	listsMu.Lock()
	defer listsMu.Unlock()

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

	if Lists[list_id] == nil {
		writeResponse(SimpleString, []string{}, true, 0, connection)
		return
	}

	if start_index < 0 {
		start_index = max(len(Lists[list_id])+start_index, 0)
	}
	if stop_index < 0 {
		stop_index = max(len(Lists[list_id])+stop_index, 0)
	}

	if start_index >= len(Lists[list_id]) {
		writeResponse(SimpleString, []string{}, true, 0, connection)
		return
	}

	if stop_index >= len(Lists[list_id]) {
		stop_index = len(Lists[list_id]) - 1
	}

	if start_index > stop_index {
		writeResponse(SimpleString, []string{}, true, 0, connection)
		return
	}

	values := []string{}

	for i := start_index; i <= stop_index; i++ {
		values = append(values, Lists[list_id][i])
	}

	writeResponse(SimpleString, values, true, stop_index-start_index+1, connection)
}

func handle_llen(connection net.Conn, list_id string) {
	listsMu.Lock()
	defer listsMu.Unlock()

	if Lists[list_id] == nil {
		writeResponse(Integer, []string{"0"}, false, 1, connection)
		return
	}

	values := []string{strconv.Itoa(len(Lists[list_id]))}
	writeResponse(Integer, values, false, 1, connection)
}

func handle_lpop(connection net.Conn, list_id string, multiple_pop bool, n int) {
	listsMu.Lock()
	defer listsMu.Unlock()

	if Lists[list_id] == nil {
		writeResponse(BulkString, []string{""}, false, 1, connection)
		return
	}
	if multiple_pop {
		size := len(Lists[list_id])
		if n > size {
			writeResponse(SimpleString, Lists[list_id], true, size, connection)
			Lists[list_id] = Lists[list_id][:0]
			return
		} else {
			writeResponse(SimpleString, Lists[list_id][0:n], true, n, connection)
			Lists[list_id] = Lists[list_id][n:]
			return
		}
	} else if len(Lists[list_id]) > 0 {
		writeResponse(SimpleString, []string{Lists[list_id][0]}, false, 1, connection)
		Lists[list_id] = Lists[list_id][1:]
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

	dbMu.RLock()
	for key := range DB {
		if reg.MatchString(key) {
			values = append(values, key)
		}
	}
	dbMu.RUnlock()

	writeResponse(SimpleString, values, true, len(values), connection)
}
