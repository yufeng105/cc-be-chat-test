package coder

import (
	"bytes"
	"encoding/binary"
	"chat-system/tcp"
	"math/rand"
	"testing"
	"time"
)

const testRound = 100

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func TestMapByte(t *testing.T) {
	clientCoder := NewClientCoder()
	svrCoder := NewServerCoder()

	for i := 0; i < testRound; i++ {
		cb := byte(rand.Int())
		cbEncoded := clientCoder.mapSendByte(cb)
		cbDecoded := svrCoder.mapRecvByte(cbEncoded)
		if cb != cbDecoded {
			t.Fatal("client->svr map byte failed")
		}
		cb2 := byte(rand.Int())
		cbEncoded2 := svrCoder.mapSendByte(cb2)
		cbDecoded2 := clientCoder.mapRecvByte(cbEncoded2)
		if cb2 != cbDecoded2 {
			t.Fatal("svr->client map byte failed")
		}
	}
}

func TestCheckCode(t *testing.T) {
	for i := 0; i < testRound; i++ {
		data := randString(rand.Intn(1000))
		checkCode := byte(0)
		for _, cb := range data {
			checkCode += byte(cb)
		}
		checkCode2 := ^checkCode + 1
		for _, cb := range data {
			checkCode2 += byte(cb)
		}
		if checkCode2 != 0 {
			t.Fatal("checkCode: ", checkCode, "checkCode2", checkCode2)
		}
	}
}

func TestCheckCodeMapped(t *testing.T) {
	clientCoder := NewClientCoder()
	svrCoder := NewServerCoder()
	for i := 0; i < testRound; i++ {
		data := randString(rand.Intn(1000))
		data2 := make([]byte, len(data))
		checkCode := byte(0)
		for i := 0; i < len(data); i++ {
			checkCode += byte(data[i])
			data2[i] = clientCoder.mapSendByte(data[i])
		}
		checkCode2 := ^checkCode + 1
		for i := 0; i < len(data2); i++ {
			cb := svrCoder.mapRecvByte(data2[i])
			checkCode2 += byte(cb)
		}
		if checkCode2 != 0 {
			t.Fatal("checkCode: ", checkCode, "checkCode2", checkCode2)
		}
	}
}

func TestSeedRandMap(t *testing.T) {
	for i := 0; i < testRound; i++ {
		seed := uint16(rand.Int())
		if seedRandMap(seed) != seedRandMap_C(seed) {
			t.Fatal("seedRandMap failed", seed, seedRandMap(seed), seedRandMap_C(seed))
		}
	}
}

func TestEncodeDecode(t *testing.T) {
	dwXorKey := rand.Uint32()
	for i := 0; i < testRound; i++ {
		data := []byte(randString(1000))
		wMainCmd := uint16(rand.Int())
		wSubCmd := uint16(rand.Int())
		packed := tcp.NewNetMsg(wMainCmd, wSubCmd, data).Buf
		
		enBuffer, enKey := encryptBuffer(packed, dwXorKey)
		deBuffer, deKey := decryptBuffer(enBuffer, dwXorKey)
		if len(packed) != len(enBuffer) || len(deBuffer) != len(packed) {
			t.Fatal("Encrypt or Decrypt len failed")
		}
		if enKey != deKey {
			t.Fatal("encrypt decrypt xor key not same")
		}
		if bytes.Compare(deBuffer, packed) != 0 {
			t.Fatal("decode packed data failed")
		}
		dwXorKey = deKey
	}
}

func TestInsertKey(t *testing.T) {
	data := []byte(randString(rand.Intn(100) + 1))
	head := []byte(randString(8))
	dataLen := uint16(len(data))
	headLen := uint16(len(head))
	WriteUint16(head[2:4], dataLen)
	key := uint32(rand.Intn(65535))
	var buf bytes.Buffer
	buf.Write(head)
	buf.Write(data)
	buffer := buf.Bytes()
	if len(buffer) != int(headLen+dataLen) {
		t.Fatal("bad copied buffer")
	}
	insertedBuf, err := insertKey(buffer, key, headLen)
	if err != nil {
		t.Fatal(err)
	}
	if len(insertedBuf) != int(headLen+dataLen+DWORD_LEN) {
		t.Fatal("bad InsertKey buffer len")
	}
	readKey, err := ReadUint32(insertedBuf[headLen : headLen+DWORD_LEN])
	if err != nil {
		t.Fatal(err)
	}
	if readKey != key {
		t.Fatal("read key failed", readKey, key)
	}
	wPacketSize, err := ReadUint16(insertedBuf[2:4])
	if err != nil {
		t.Fatal(err)
	}
	if wPacketSize != dataLen+DWORD_LEN {
		t.Fatal("read wPacketSize failed")
	}
	if bytes.Compare(data, insertedBuf[headLen+DWORD_LEN:headLen+dataLen+DWORD_LEN]) != 0 {
		t.Fatal("Compare data failed")
	}
}

func TestExtractKey(t *testing.T) {
	data := []byte(randString(rand.Intn(100) + 1))
	head := []byte(randString(8))
	dataLen := uint16(len(data))
	headLen := uint16(len(head))
	WriteUint16(head[2:4], dataLen)
	key := uint32(rand.Intn(65535))
	buf := &bytes.Buffer{}
	buf.Write(head)
	binary.Write(buf, binary.LittleEndian, key)
	buf.Write(data)
	buffer := buf.Bytes()
	if len(buffer) != int(headLen+dataLen+DWORD_LEN) {
		t.Fatal("bad copied buffer")
	}
	extracedBuf, extractedKey, err := extractKey(buffer, headLen)
	if err != nil {
		t.Fatal(err)
	}
	if len(extracedBuf) != int(len(buffer)-DWORD_LEN) {
		t.Fatal("bad extracedBuf buffer len")
	}
	if extractedKey != key {
		t.Fatal("read key failed", extractedKey, key)
	}
	wPacketSize, err := ReadUint16(extracedBuf[2:4])
	if err != nil {
		t.Fatal(err)
	}
	if wPacketSize != dataLen-DWORD_LEN {
		t.Fatal("read wPacketSize failed")
	}
	if bytes.Compare(data, extracedBuf[headLen:]) != 0 {
		t.Fatal("Compare data failed")
	}
}

func checkHead(round int, t *testing.T, decodedBytes []byte, wMainCmd, wSubCmd uint16) {
	msg := tcp.NetMsg{Buf: decodedBytes}
	if msg.MainCmd() != wMainCmd {
		t.Fatal("round", round, "decode wMainCmd failed", "source:", wMainCmd, "decoded:", msg.MainCmd())
	}
	if msg.SubCmd() != wSubCmd {
		t.Fatal("round", round, "decode wSubCmd failed", "source:", wSubCmd, "decoded:", msg.SubCmd())
	}
}

func checkBody(round int, t *testing.T, decodedBytes []byte, body []byte) {
	decodedBody := decodedBytes[SizeOf_CMD_Head:]
	if bytes.Compare(body, decodedBody) != 0 {
		t.Fatal("round", round, "check decoded body failed", "decoded:", decodedBody, "source", body)
	}
}

func TestCoder1(t *testing.T) {
	clientCoder := NewClientCoder()
	svrCoder := NewServerCoder()

	for i := 0; i < testRound; i++ {
		data := randString(rand.Intn(10) + 1)
		wMainCmd := uint16(rand.Intn(1000) + 1)
		wSubCmd := uint16(rand.Intn(1000) + 1)
		msgCli := tcp.NewNetMsg(wMainCmd, wSubCmd, []byte(data))
		client_encoded, err := msgCli.Encrypt(clientCoder)
		if err != nil {
			t.Fatal(i, err)
		}
		
		msgSvr := &tcp.NetMsg{Buf: client_encoded}
		err = msgSvr.Decrypt(svrCoder)
		if err != nil {
			t.Fatal(i, err)
		}
		svr_decoded := msgSvr.Buf
		checkHead(i, t, svr_decoded, wMainCmd, wSubCmd)
		checkBody(i, t, svr_decoded, []byte(data))

		msgSvr = tcp.NewNetMsg(wMainCmd, wSubCmd, svr_decoded[SizeOf_CMD_Head:])
		svr_encoded, err := msgSvr.Encrypt(svrCoder)
		if err != nil {
			t.Fatal(err)
		}
		
		msgCli = &tcp.NetMsg{Buf: svr_encoded}
		err = msgCli.Decrypt(clientCoder)
		if err != nil {
			t.Fatal(err)
		}
		client_decoded := msgCli.Buf
		checkHead(i, t, client_decoded, wMainCmd, wSubCmd)
		checkBody(i, t, client_decoded, []byte(data))
	}
}

func TestCoder2(t *testing.T) {
	clientCoder := NewClientCoder()
	svrCoder := NewServerCoder()

	for i := 0; i < testRound; i++ {
		data := randString(rand.Intn(10) + 1)
		wMainCmd := uint16(rand.Intn(1000) + 1)
		wSubCmd := uint16(rand.Intn(1000) + 1)
		msgCli := tcp.NewNetMsg(wMainCmd, wSubCmd, []byte(data))
		client_encoded, err := msgCli.Encrypt(clientCoder)
		if err != nil {
			t.Fatal(i, err)
		}
		
		msgSvr := &tcp.NetMsg{Buf: client_encoded}
		err = msgSvr.Decrypt(svrCoder)
		if err != nil {
			t.Fatal(i, err)
		}
		svr_decoded := msgSvr.Buf
		checkHead(i, t, svr_decoded, wMainCmd, wSubCmd)
		checkBody(i, t, svr_decoded, []byte(data))
	}
}
