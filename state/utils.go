package state

import (
	"bytes"
	"encoding/gob"
	"errors"
)

func encodeOutpoints(outpoints []Outpoint) ([]byte, error) {
	if outpoints == nil {
		return nil, nil
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(outpoints); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decodeOutpoints(data []byte) ([]Outpoint, error) {
	if data == nil {
		return nil, nil
	}

	if len(data) == 0 {
		return nil, errors.New("expect non-empty bytes")
	}

	decoder := gob.NewDecoder(bytes.NewReader(data))
	var outpoints []Outpoint
	if err := decoder.Decode(&outpoints); err != nil {
		return nil, err
	}

	return outpoints, nil
}
