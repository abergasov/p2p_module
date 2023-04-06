package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"

	atomicfile "github.com/natefinch/atomic"
)

// AtomicallySaveToFile saves the given data to the given file atomically. Append a checksum to the data before writing it
func AtomicallySaveToFile(path string, data []byte) error {
	checkSum := crc32.Checksum(data, crc32.MakeTable(crc32.IEEE))
	checkSumBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(checkSumBytes, checkSum)
	resultData := append(checkSumBytes, data...)

	if err := atomicfile.WriteFile(path, bytes.NewReader(resultData)); err != nil {
		return fmt.Errorf("failed to write addresses to file: %w", err)
	}
	return nil
}

// LoadFromFile loads the data from the given file and verifies the checksum. It returns the data without the checksum
func LoadFromFile(path string) (data []byte, err error) {
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %w", err)
	}
	r, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer r.Close()

	data, err = io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	if len(data) < 4 {
		return nil, errors.New("file is too short")
	}
	fileCrc := binary.LittleEndian.Uint32(data[:4])
	dataCrc := crc32.Checksum(data[4:], crc32.MakeTable(crc32.IEEE))
	if fileCrc != dataCrc {
		return nil, fmt.Errorf("checksum mismatch: %x != %x", fileCrc, dataCrc)
	}
	return data[4:], nil
}
