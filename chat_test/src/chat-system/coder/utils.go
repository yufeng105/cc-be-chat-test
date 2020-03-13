package coder

import (
	"bytes"
	"encoding/binary"
	"errors"
)

func insertKey(cbDataBuffer []byte, dwSendXorKey uint32, wHeadLen uint16) (newDataBuffer []byte, err error) {
	if len(cbDataBuffer) < int(wHeadLen) {
		err = errors.New("cbDataBuffer is too small")
		return
	}
	wPacketSize := uint16(0)
	if wPacketSize, err = ReadUint16(cbDataBuffer[2:4]); err != nil {
		return
	}
	buffer := &bytes.Buffer{}
	if _, err = buffer.Write(cbDataBuffer[:2]); err != nil {
		return
	}
	if err = binary.Write(buffer, binary.LittleEndian, wPacketSize+DWORD_LEN); err != nil {
		return
	}
	if _, err = buffer.Write(cbDataBuffer[4:wHeadLen]); err != nil {
		return
	}
	if err = binary.Write(buffer, binary.LittleEndian, dwSendXorKey); err != nil {
		return
	}
	if _, err = buffer.Write(cbDataBuffer[wHeadLen:]); err != nil {
		return
	}
	newDataBuffer = buffer.Bytes()
	return
}

func extractKey(cbDataBuffer []byte, wHeadLen uint16) (newDataBuffer []byte, dwRecvXorKey uint32, err error) {
	if len(cbDataBuffer) < int(wHeadLen+DWORD_LEN) {
		err = errors.New("cbDataBuffer too small")
		return
	}
	if dwRecvXorKey, err = ReadUint32(cbDataBuffer[wHeadLen : wHeadLen+DWORD_LEN]); err != nil {
		return
	}
	wPacketSize := uint16(0)
	if wPacketSize, err = ReadUint16(cbDataBuffer[2:4]); err != nil {
		return
	}
	buffer := &bytes.Buffer{}
	if _, err = buffer.Write(cbDataBuffer[:2]); err != nil {
		return
	}
	if err = binary.Write(buffer, binary.LittleEndian, wPacketSize-DWORD_LEN); err != nil {
		return
	}
	if _, err = buffer.Write(cbDataBuffer[4:wHeadLen]); err != nil {
		return
	}
	if _, err = buffer.Write(cbDataBuffer[wHeadLen+DWORD_LEN:]); err != nil {
		return
	}
	newDataBuffer = buffer.Bytes()
	return
}

func seedRandMap(wSeed uint16) uint16 {
	dwHold := uint32(wSeed)
	dwHold = dwHold*241103 + 2533101
	return uint16(dwHold >> 16)
}

func seedRandMap_C(wSeed uint16) uint16 {
	return uint16(SeedRandMap(wSeed))
}

func encryptBuffer(cbDataBuffer []byte, dwXorKey uint32) (cbNewDataBuffer []byte, dwNewXorKey uint32) {
	//调整长度
	wDataSize := uint16(len(cbDataBuffer))
	wEncryptSize := uint16(wDataSize - SizeOf_CMD_Info)
	wSnapCount := uint16(0)
	if (wEncryptSize % DWORD_LEN) != 0 {
		wSnapCount = DWORD_LEN - wEncryptSize%DWORD_LEN
		bsSnap := make([]byte, wSnapCount)
		cbDataBuffer = append(cbDataBuffer, bsSnap...)
	}
	dwNewXorKey = uint32(c_encryptBuffer(
		cbDataBuffer,
		dwXorKey,
		wEncryptSize,
		wSnapCount))
	cbNewDataBuffer = cbDataBuffer[:wDataSize]
	return
}

func decryptBuffer(cbDataBuffer []byte, dwXorKey uint32) (cbNewDataBuffer []byte, dwNewXorKey uint32) {
	//调整长度
	wDataSize := uint16(len(cbDataBuffer))
	wEncryptSize := uint16(wDataSize - SizeOf_CMD_Info)
	wSnapCount := uint16(0)
	if (wEncryptSize % DWORD_LEN) != 0 {
		wSnapCount = DWORD_LEN - wEncryptSize%DWORD_LEN
		bsSnap := make([]byte, wSnapCount)
		cbDataBuffer = append(cbDataBuffer, bsSnap...)
	}
	dwNewXorKey = uint32(c_decryptBuffer(
		cbDataBuffer,
		dwXorKey,
		wEncryptSize,
		wSnapCount,
		wDataSize))
	cbNewDataBuffer = cbDataBuffer[:wDataSize]
	return
}
