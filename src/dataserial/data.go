package dataserial

import (
	"bytes"
	"encoding/gob"
)

// RPCdata represents the serializing format of structured data
type RPCdata struct {
	Name string        // name of the function
	Args []interface{} // request's or response's body expect error.
	Err  string        // Error any executing remote server
}

// Encode The RPCdata in binary format which can
// be sent over the network.
func Encode(data RPCdata) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode the binary data into the Go RPC struct
func Decode(b []byte) (RPCdata, error) {
	buf := bytes.NewBuffer(b)
	decoder := gob.NewDecoder(buf)
	var data RPCdata
	if err := decoder.Decode(&data); err != nil {
		return RPCdata{}, err
	}
	return data, nil
}