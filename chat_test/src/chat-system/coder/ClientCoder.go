package coder

import (
	"fmt"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type ClientCoder struct {
	m_cbSendRound       byte   //字节映射
	m_cbRecvRound       byte   //字节映射
	m_dwSendXorKey      uint32 //发送密钥
	m_dwRecvXorKey      uint32 //接收密钥
	m_dwSendPacketCount uint32 //发送计数
	m_dwRecvPacketCount uint32 //接受计数
}

func NewClientCoder() *ClientCoder {
	return &ClientCoder{}
}

//deprecated
func (cc *ClientCoder) Encode(wMainCmd, wSubCmd uint16, data interface{}) (encoded []byte, err error) {
	return
}

//cbDataBuffer = with head
func (cc *ClientCoder) Encrypt(cbDataBuffer []byte) (encoded []byte, err error) {
	wDataSize := uint16(len(cbDataBuffer))
	if wDataSize < SizeOf_CMD_Head {
		err = fmt.Errorf("client encrypt wDataSize (%d) too short", wDataSize)
		return
	}
	if wDataSize > SizeOf_CMD_Head+SOCKET_PACKET {
		err = fmt.Errorf("client encrypt wDataSize(%d) too long", wDataSize)
		return
	}

	cbCheckCode := byte(0)
	for i := uint16(SizeOf_CMD_Info); i < wDataSize; i++ {
		cbCheckCode += cbDataBuffer[i]
		cbDataBuffer[i] = cc.mapSendByte(cbDataBuffer[i])
	}
	cbDataBuffer[1] = ^cbCheckCode + 1

	if err = WriteUint16(cbDataBuffer[2:4], wDataSize); err != nil {
		return
	}

	if cc.m_dwSendPacketCount == 0 {
		dwXorKey := uint32(rand.Int31())
		//映射种子
		dwXorKey = uint32(SeedRandMap(uint16(dwXorKey)))
		dwXorKey = dwXorKey | uint32(SeedRandMap(uint16(dwXorKey>>16))<<16)
		dwXorKey = dwXorKey ^ uint32(g_dwPacketKey)

		cc.m_dwSendXorKey = dwXorKey
		cc.m_dwRecvXorKey = dwXorKey
	}

	dwNewSendXorKey := uint32(0)
	encoded, dwNewSendXorKey = encryptBuffer(cbDataBuffer, cc.m_dwSendXorKey)

	//插入密钥
	if cc.m_dwSendPacketCount == 0 {
		if encoded, err = insertKey(encoded, cc.m_dwSendXorKey, SizeOf_CMD_Head); err != nil {
			return
		}
	}

	//设置变量
	cc.m_dwSendPacketCount++
	cc.m_dwSendXorKey = dwNewSendXorKey
	return
}

func (cc *ClientCoder) Decrypt(cbDataBuffer []byte) (decoded []byte, err error) {
	wDataSize := uint16(len(cbDataBuffer))
	if cc.m_dwSendPacketCount <= 0 {
		err = fmt.Errorf("client decrypt haven't sent any packet")
		return
	}
	if wDataSize < SizeOf_CMD_Head {
		err = fmt.Errorf("client decrypt wDataSize(%d) is too short", wDataSize)
		return
	}
	wPacketSize := uint16(0)
	if wPacketSize, err = ReadUint16(cbDataBuffer[2:4]); err != nil {
		return
	}
	if wPacketSize != wDataSize {
		err = fmt.Errorf("client decrypt wPacketSize(%d) != wDataSize(%d)", wPacketSize, wDataSize)
		return
	}

	dwNewRecvXorKey := uint32(0)
	decoded, dwNewRecvXorKey = decryptBuffer(cbDataBuffer, cc.m_dwRecvXorKey)

	cbCheckCode := byte(decoded[1])
	for i := uint16(SizeOf_CMD_Info); i < wDataSize; i++ {
		decoded[i] = cc.mapRecvByte(decoded[i])
		cbCheckCode += decoded[i]
	}
	if cbCheckCode != 0 {
		err = fmt.Errorf("client decrypt check code (%d) failed", cbCheckCode)
		return
	}

	cc.m_dwRecvPacketCount++
	cc.m_dwRecvXorKey = dwNewRecvXorKey
	return
}

func (cc *ClientCoder) mapSendByte(cb byte) byte {
	cbMap := g_SendByteMap[byte(cb+cc.m_cbSendRound)]
	cc.m_cbSendRound += 3
	return cbMap
}

func (cc *ClientCoder) mapRecvByte(cb byte) byte {
	cbMap := g_RecvByteMap[cb] - cc.m_cbRecvRound
	cc.m_cbRecvRound += 3
	return cbMap
}
