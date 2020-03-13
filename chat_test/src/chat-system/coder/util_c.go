package coder

import (
	"encoding/binary"
	"unsafe"
)

const g_dwPacketKey uint32 = 0x1FBFEF1F
const SizeOf_CMD_Info = 0x4
const SizeOf_CMD_Command = 0x4
const SizeOf_CMD_Head = 0x8

func SeedRandMap(wSeed uint16) uint16 {
	dwHold := uint32(wSeed)
	return uint16((dwHold*241103 + 2533101) >> 16)
}

func c_encryptBuffer(cbDataBuffer []byte, dwXorKey uint32, wEncryptSize uint16, wSnapCount uint16) uint32 {
	bufer := cbDataBuffer[SizeOf_CMD_Info:]
	pwSeed := 0
	pdwXor := 0
	wEncrypCount := (wEncryptSize + wSnapCount) / 4
	for i := uint16(0); i < wEncrypCount; i++ {
		*(*uint32)(unsafe.Pointer(&bufer[pdwXor])) ^= dwXorKey
		pdwXor += 4

		dwXorKey = uint32(SeedRandMap(*(*uint16)(unsafe.Pointer(&bufer[pwSeed]))))
		pwSeed += 2
		dwXorKey |= uint32(SeedRandMap(*(*uint16)(unsafe.Pointer(&bufer[pwSeed])))) << 16
		pwSeed += 2
		dwXorKey ^= g_dwPacketKey
	}
	return dwXorKey
}

func c_decryptBuffer(pcbDataBuffer []byte, dwRecvXorKey uint32, wEncryptSize uint16, wSnapCount uint16, wDataSize uint16) uint32 {
	//解密数据
	bufer := pcbDataBuffer[SizeOf_CMD_Info:]
	dwXorKey := dwRecvXorKey
	pwSeed := 0
	pdwXor := 0
	wEncrypCount := (wEncryptSize + wSnapCount) / 4
	for i := uint16(0); i < wEncrypCount; i++ {
		if i == (wEncrypCount-1) && wSnapCount > 0 {
			buf := make([]byte, 4)
			binary.LittleEndian.PutUint32(buf, dwRecvXorKey)
			copy(pcbDataBuffer[wDataSize:], buf[4-wSnapCount:])
		}
		dwXorKey = uint32(SeedRandMap(*(*uint16)(unsafe.Pointer(&bufer[pwSeed]))))
		pwSeed += 2
		dwXorKey |= uint32(SeedRandMap(*(*uint16)(unsafe.Pointer(&bufer[pwSeed])))) << 16
		pwSeed += 2
		dwXorKey ^= g_dwPacketKey
		*(*uint32)(unsafe.Pointer(&bufer[pdwXor])) ^= dwRecvXorKey
		pdwXor += 4
		dwRecvXorKey = dwXorKey
	}

	return dwRecvXorKey
}
