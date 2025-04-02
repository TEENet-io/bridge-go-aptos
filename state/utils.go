package state

import (
	"bytes"
	"encoding/gob"
	"errors"

	"github.com/TEENet-io/bridge-go/agreement"
)

func EncodeOutpoints(outpoints []agreement.BtcOutpoint) ([]byte, error) {
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

func DecodeOutpoints(data []byte) ([]agreement.BtcOutpoint, error) {
	if data == nil {
		return nil, nil
	}

	if len(data) == 0 {
		return nil, errors.New("expect non-empty bytes")
	}

	decoder := gob.NewDecoder(bytes.NewReader(data))
	var outpoints []agreement.BtcOutpoint
	if err := decoder.Decode(&outpoints); err != nil {
		return nil, err
	}

	return outpoints, nil
}
