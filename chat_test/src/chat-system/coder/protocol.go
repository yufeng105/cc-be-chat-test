package coder

import (
	"bytes"
	"encoding/binary"
)

func Unpack(src []byte, unpacked interface{}) (err error) {
	err = binary.Read(bytes.NewReader(src), binary.LittleEndian, unpacked)
	return
}

func ReadUint16(data []byte) (uint16, error) {
	var ret uint16
	err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &ret)
	return ret, err
}

func ReadUint32(data []byte) (uint32, error) {
	var ret uint32
	err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &ret)
	return ret, err
}

func WriteUint16(dst []byte, val uint16) error {
	buf := bytes.NewBuffer(dst)
	buf.Reset()
	return binary.Write(buf, binary.LittleEndian, val)
}

func WriteVal(dst []byte, val interface{}) error {
	buf := bytes.NewBuffer(dst)
	buf.Reset()
	return binary.Write(buf, binary.LittleEndian, val)
}
