package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
)

func loadRDBFile(path string) ([]byte, error) {
	log.Println("loadRDBFile", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func readLength(data []byte, pos *int) (int, error) {

	if *pos >= len(data) {
		return 0, fmt.Errorf("unexpected end of data while reading length")
	}

	first := data[*pos]
	(*pos)++
	switch first >> 6 {
	case 0b00:
		return int(first & 0x3F), nil
	case 0b01:
		if *pos >= len(data) {
			return 0, fmt.Errorf("unexpected end of data while reading 14 bit length")
		}
		second := data[*pos]
		(*pos)++
		return int(((first & 0x3F) << 8) | (second)), nil
	case 0b10:

		if *pos+4 > len(data) {
			return 0, fmt.Errorf("unexpected end of data while reading 32 bit length")
		}
		val := binary.BigEndian.Uint32(data[*pos : (*pos)+4])
		*pos += 4
		return int(val), nil
	default:
		return 0, fmt.Errorf("unsupported string encoding byte")

	}
}

func readHeader(data []byte, pos *int) {

	log.Println("trigger ReadHeader command:", string(data[*pos:*pos+9]))
	*pos += 9
}

func parseMetadataSection(data []byte, pos *int) (map[string]string, error) {
	log.Println("starting parse MetaData......")
	metadata := make(map[string]string)

	for *pos < len(data) {
		b := data[*pos]

		if b != 0xFA {

			break
		}

		(*pos)++

		key, err := readOneEncodedString(data, pos)
		if err != nil {
			return nil, err
		}

		value, err := readOneEncodedString(data, pos)
		if err != nil {
			return nil, err
		}

		metadata[key] = value
	}

	return metadata, nil
}

func parseDatabaseSection(data []byte, pos *int) (map[string]*Entry, error) {
	log.Println("start parsing database section")
	db := make(map[string]*Entry)

	for *pos < len(data) {
		prefix := data[*pos]
		log.Println("Prefix is", fmt.Sprintf("0x%X", prefix))

		if prefix == 0xFF {
			log.Println("0xFF prefix found")
			break
		}

		if prefix == 0xFE {
			log.Println("0xFE prefix found")

			(*pos)++
			_, err := readLength(data, pos)
			if err != nil {
				return nil, err
			}

			continue
		}

		if prefix == 0xFB {
			log.Println("0xFB prefix found")

			(*pos)++

			keyLength, err := readLength(data, pos)
			if err != nil {
				return nil, err
			}

			valLength, err := readLength(data, pos)
			if err != nil {
				return nil, err
			}

			log.Println("hash table size:", keyLength, valLength)
			continue
		}

		if prefix == 0xFC {
			log.Println("0xFC prefix found - expiry in milliseconds")

			(*pos)++
			*pos += 8

			continue
		}

		if prefix == 0xFD {
			log.Println("0xFD prefix found - expiry in seconds")

			(*pos)++
			*pos += 4

			continue
		}

		if prefix == 0x00 {
			log.Println("0x00 prefix found - string key/value")

			(*pos)++

			key, err := readOneEncodedString(data, pos)
			if err != nil {
				return nil, err
			}

			val, err := readOneEncodedString(data, pos)
			if err != nil {
				return nil, err
			}

			log.Println("key, value:", key, val)

			db[key] = &Entry{
				Type:  StringType,
				Value: val,
			}
			continue
		}

		return nil, fmt.Errorf("unsupported RDB prefix: 0x%X at position %d", prefix, *pos)
	}

	return db, nil
}

func readOneEncodedString(data []byte, pos *int) (string, error) {
	if *pos > len(data) {
		return "", fmt.Errorf("unexpected end of data while reading string")
	}
	length, err := readLength(data, pos)

	if err != nil {
		return "", err
	}
	if *pos+length > len(data) {
		return "", fmt.Errorf("not enough bytes to read a string")
	}
	s := string(data[*pos : (*pos)+length])

	log.Println("read one Encoded string:", s)
	*pos = *pos + length
	log.Println("current postition", int(*pos))
	return s, nil
}

func loadRDBIntoStore(config Config) error {

	path := config.dir + "/" + config.dbfilename

	data, err := loadRDBFile(path)
	if err != nil {
		return err
	}

	pos := 0

	readHeader(data, &pos)

	_, err = parseMetadataSection(data, &pos)
	if err != nil {
		return err
	}

	db, err := parseDatabaseSection(data, &pos)
	if err != nil {
		return err
	}

	DB.Lock()
	defer DB.Unlock()

	for k, v := range db {
		DB.Data[k] = v
	}

	return nil
}
