package coder

import (
	"fmt"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type ServerCoder struct {
	m_cbSendRound       byte   //字节映射
	m_cbRecvRound       byte   //字节映射
	m_dwSendXorKey      uint32 //发送密钥
	m_dwRecvXorKey      uint32 //接收密钥
	m_dwSendPacketCount uint32 //发送计数
	m_dwRecvPacketCount uint32 //接受计数
}

func NewServerCoder() *ServerCoder {
	return &ServerCoder{}
}

//deprecated
func (sc *ServerCoder) Encode(wMainCmd, wSubCmd uint16, data interface{}) (encoded []byte, err error) {
	return
}

//wDataSize = with head
func (sc *ServerCoder) Encrypt(cbDataBuffer []byte) (encoded []byte, err error) {
	wDataSize := uint16(len(cbDataBuffer))
	if wDataSize < SizeOf_CMD_Head {
		err = fmt.Errorf("svr encrypt wDataSize(%d) too short", wDataSize)
		return
	}
	if wDataSize > SizeOf_CMD_Head+SOCKET_PACKET {
		err = fmt.Errorf("svr encrypt wDataSize(%d) too long", wDataSize)
		return
	}

	if sc.m_dwSendXorKey == 0 {
		err = fmt.Errorf("svr dwSendXorKey is null")
		return
	}

	cbCheckCode := byte(0)
	for i := uint16(SizeOf_CMD_Info); i < wDataSize; i++ {
		cbCheckCode += cbDataBuffer[i]
		cbDataBuffer[i] = sc.mapSendByte(cbDataBuffer[i])
	}

	cbDataBuffer[1] = ^cbCheckCode + 1
	if err = WriteUint16(cbDataBuffer[2:4], wDataSize); err != nil {
		return
	}

	dwNewSendXorKey := uint32(0)
	encoded, dwNewSendXorKey = encryptBuffer(cbDataBuffer, sc.m_dwSendXorKey)

	//设置变量
	sc.m_dwSendPacketCount++
	sc.m_dwSendXorKey = dwNewSendXorKey
	return
}

func (sc *ServerCoder) Decrypt(cbDataBuffer []byte) (decoded []byte, err error) {
	wDataSize := uint16(len(cbDataBuffer))
	if wDataSize < SizeOf_CMD_Head {
		err = fmt.Errorf("svr decrypt wDataSize(%d) is too short", wDataSize)
		return
	}
	wPacketSize := uint16(0)
	if wPacketSize, err = ReadUint16(cbDataBuffer[2:4]); err != nil {
		return
	}
	if wPacketSize != wDataSize {
		err = fmt.Errorf("svr decrypt wPacketSize(%d) != wDataSize(%d)", wPacketSize, wDataSize)
		return
	}

	if sc.m_dwRecvPacketCount == 0 {
		dwRecvXorKey := uint32(0)
		if cbDataBuffer, dwRecvXorKey, err = extractKey(cbDataBuffer, SizeOf_CMD_Head); err != nil {
			return
		}
		sc.m_dwRecvXorKey = dwRecvXorKey
		sc.m_dwSendXorKey = dwRecvXorKey
	}

	dwNewRecvXorKey := uint32(0)
	decoded, dwNewRecvXorKey = decryptBuffer(cbDataBuffer, sc.m_dwRecvXorKey)

	cbCheckCode := byte(decoded[1])
	for i := SizeOf_CMD_Info; i < len(decoded); i++ {
		decoded[i] = sc.mapRecvByte(decoded[i])
		cbCheckCode += decoded[i]
	}
	if cbCheckCode != 0 {
		err = fmt.Errorf("svr check code (%d) failed", cbCheckCode)
		return
	}
	sc.m_dwRecvPacketCount++
	sc.m_dwRecvXorKey = dwNewRecvXorKey
	return
}

func (sc *ServerCoder) mapSendByte(cb byte) byte {
	cbMap := g_SendByteMap[byte(cb+sc.m_cbSendRound)]
	sc.m_cbSendRound += 3
	return cbMap
}

func (sc *ServerCoder) mapRecvByte(cb byte) byte {
	cbMap := g_RecvByteMap[cb] - sc.m_cbRecvRound
	sc.m_cbRecvRound += 3
	return cbMap
}
