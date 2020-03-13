package tcp

import (
	"errors"
	"fmt"
	"chat-system/log"
	"chat-system/timer"
	"chat-system/util"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type ITNetDelegate interface {
	Init()

	SetServer(*TcpServer)
	OnServerStop()
	HandleServerStop(func())

	NewCoder(func() Coder)
	NewTcpClient(conn *net.TCPConn, idx int) ITcpClient
	OnConnect(ITcpClient)
	HandleConnect(func(ITcpClient))

	OnDisconnected(ITcpClient)
	HandleDisconnected(func(ITcpClient))

	RecvMsg(ITcpClient) INetMsg
	SendMsg(ITcpClient, INetMsg) error
	SendData(ITcpClient, []byte) error
	HandleMsg(ITcpClient, INetMsg)

	Handle(mainCmd uint16, subCmd uint16, cb func(client ITcpClient, msg INetMsg))
	// Unhandle(mainCmd uint16, subCmd uint16)

	GetRecvBlockTime() time.Duration
	SetRecvBlockTime(time.Duration)

	GetSendBlockTime() time.Duration
	SetSendBlockTime(time.Duration)

	GetRecvBufLen() int
	SetRecvBufLen(int)

	GetSendBufLen() int
	SetSendBufLen(int)

	GetSendQueueSize() int
	SetSendQueueSize(int)

	GetMaxPackLen() int
	SetMaxPackLen(int)

	IsNoDelay() bool
	SetNoDelay(bool)

	IsKeepAlive() bool
	SetKeepAlive(bool)

	GetAliveTime() time.Duration
	SetAliveTime(time.Duration)

	GetTimer() *timer.Timer
}

type DefaultNetDelegate struct {
	sync.Mutex
	//Mutex
	inited     int32
	Server     *TcpServer
	handlerMap map[uint32]func(client ITcpClient, msg INetMsg)

	maxPackLen    int
	recvBlockTime time.Duration
	recvBufLen    int
	sendBlockTime time.Duration
	sendBufLen    int
	sendQueueSize int
	aliveTime     time.Duration
	noDelay       bool
	keepAlive     bool
	delegate      ITNetDelegate
	tag           string
	asynctimer    *timer.Timer
	newCoder      func() Coder

	onServerStop func()

	onConnect    func(ITcpClient)
	onDisconnect func(ITcpClient)
}

func (dele *DefaultNetDelegate) Init() {
	if atomic.CompareAndSwapInt32(&(dele.inited), 0, 1) {
		if dele.GetAliveTime() == 0 {
			dele.SetAliveTime(DEFAULT_KEEP_ALIVE_TIME)
		}

		if dele.GetRecvBlockTime() == 0 {
			dele.SetRecvBlockTime(DEFAULT_RECV_BLOCK_TIME)
		}

		if dele.GetSendBlockTime() == 0 {
			dele.SetSendBlockTime(DEFAULT_SEND_BLOCK_TIME)
		}

		if dele.GetMaxPackLen() == 0 {
			dele.SetMaxPackLen(DEFAULT_MAX_PACK_LEN)
		}

		if dele.GetRecvBufLen() == 0 {
			dele.SetRecvBufLen(DEFAULT_RECV_BUF_LEN)
		}
		if dele.GetSendBufLen() == 0 {
			dele.SetSendBufLen(DEFAULT_SEND_BUF_LEN)
		}

		if dele.GetSendQueueSize() == 0 {
			dele.SetSendQueueSize(DEFAULT_SEND_Q_SIZE)
		}

		if dele.asynctimer == nil {
			dele.asynctimer = timer.NewTimer()
		}

		dele.tag = "DefaultNetDelegate"
	}
}

func (dele *DefaultNetDelegate) SetServer(server *TcpServer) {
	dele.Lock()
	defer dele.Unlock()
	dele.Server = server
}

func (dele *DefaultNetDelegate) OnServerStop() {
	if dele.onServerStop != nil {
		dele.onServerStop()
	}
}

func (dele *DefaultNetDelegate) HandleServerStop(h func()) {
	dele.onServerStop = h
}

func (dele *DefaultNetDelegate) NewCoder(f func() Coder) {
	dele.newCoder = f
}

func (dele *DefaultNetDelegate) NewTcpClient(conn *net.TCPConn, idx int) ITcpClient {
	var coder Coder = nil
	if dele.newCoder != nil {
		coder = dele.newCoder()
	}
	return newTcpClient(dele, conn, idx, coder)
}

func (dele *DefaultNetDelegate) OnConnect(client ITcpClient) {
	if dele.onConnect != nil {
		dele.onConnect(client)
	}
}

func (dele *DefaultNetDelegate) HandleConnect(h func(ITcpClient)) {
	dele.onConnect = h
}

func (dele *DefaultNetDelegate) OnDisconnected(client ITcpClient) {
	if dele.onDisconnect != nil {
		dele.onDisconnect(client)
	}
}

func (dele *DefaultNetDelegate) HandleDisconnected(h func(ITcpClient)) {
	dele.onDisconnect = h
}

func (dele *DefaultNetDelegate) RecvMsg(client ITcpClient) INetMsg {
	pkt := struct {
		err     error
		msg     *NetMsg
		readLen int
		packLen int
	}{
		err: nil,
		msg: &NetMsg{
			//Client: client,
			Buf: make([]byte, DEFAULT_PACK_HEAD_LEN),
		},
		readLen: 0,
		packLen: 0,
	}

	if pkt.err = (*client.GetConn()).SetReadDeadline(time.Now().Add(dele.recvBlockTime)); pkt.err != nil {
		log.Debug("%s RecvMsg SetReadDeadline Err: %v.", client.Info(), pkt.err)
		goto Exit
	}

	pkt.readLen, pkt.err = io.ReadFull(client.GetConn(), pkt.msg.Buf[:4])
	client.AddReadLen(pkt.readLen)
	if pkt.err != nil || pkt.readLen < 4 {
		//log.Debug("%s RecvMsg Read Head Err: %v %d.", client.Info(), pkt.err, pkt.readLen)
		goto Exit
	}

	if pkt.msg.Buf[0] != 0x89 && pkt.msg.Buf[0] != 0x99 {
		log.Debug("%s 数据包不匹配", client.Info())
		goto Exit
	}
	//log.Debug("%s RecvMsg Read Head : %d.", client.Info(), pkt.msg.Buf[0])
	if pkt.msg.Buf[0] == 0x99 {
		pkt.msg.Buf = append(pkt.msg.Buf, make([]byte, 4)...)
	}

	pkt.readLen, pkt.err = io.ReadFull(client.GetConn(), pkt.msg.Buf[4:])
	if pkt.err != nil || pkt.readLen < len(pkt.msg.Buf)-4 {
		log.Debug("%s RecvMsg Read Head Err: %v %d.", client.Info(), pkt.err, pkt.readLen)
		goto Exit
	}

	pkt.packLen = int(pkt.msg.PackLen())
	if pkt.msg.Buf[0] == 0x99 {
		ip := pkt.msg.Buf[4:8]
		client.SetRealIP(fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3]))
		pkt.msg.Buf = append(pkt.msg.Buf[:4], pkt.msg.Buf[8:]...)
		// pkt.packLen -= 4
		pkt.msg.SetPackLen(uint16(pkt.packLen))
		pkt.msg.Buf[0] = 137
	}

	if pkt.packLen > DEFAULT_PACK_HEAD_LEN {
		if pkt.err = (*client.GetConn()).SetReadDeadline(time.Now().Add(dele.recvBlockTime)); pkt.err != nil {
			log.Debug("%s RecvMsg SetReadDeadline Err: %v.", client.Info(), pkt.err)
			goto Exit
		}

		if pkt.packLen > dele.maxPackLen {
			log.Debug("%s RecvMsg Read Body Err: Msg Len(%d) > MAXPACK_LEN(%d)", client.Info(), pkt.packLen, dele.maxPackLen)
			goto Exit
		}
		pkt.msg.Buf = append(pkt.msg.Buf, make([]byte, pkt.packLen-DEFAULT_PACK_HEAD_LEN)...)
		pkt.readLen, pkt.err = io.ReadFull(client.GetConn(), pkt.msg.Buf[DEFAULT_PACK_HEAD_LEN:])
		client.AddReadLen(pkt.readLen)
		if pkt.err != nil {
			log.Debug("%s RecvMsg Read Body Err: %v.", client.Info(), pkt.err)
			goto Exit
		}
	}

	// log.Debug("RecvMsg: %d %d %d", pkt.msg.MainCmd(), pkt.msg.SubCmd(), pkt.msg.PackLen())
	if pkt.err = pkt.msg.Decrypt(client.Coder()); pkt.err != nil {
		log.Debug("%s RecvMsg Decrypt Err: %v.", client.Info(), pkt.err)
		return nil
	}
	// if !pkt.msg.Decrypt() {
	// 	goto Exit
	// }
	return pkt.msg

Exit:
	return nil
}

func (dele *DefaultNetDelegate) SendMsg(client ITcpClient, msg INetMsg) error {
	pkt := struct {
		err      error
		buf      []byte
		totalLen int
	}{}

	// if !msg.Encrypt() {
	// 	return false
	// }

	if pkt.buf, pkt.err = msg.Encrypt(client.Coder()); pkt.err != nil {
		log.Debug("%s RecvMsg Encrypt Err: %v.", client.Info(), pkt.err)
		return pkt.err
	}
	// log.Debug("SendMsg: %d %d %d", msg.MainCmd(), msg.SubCmd(), msg.PackLen())

	pkt.totalLen = len(pkt.buf)

	if pkt.totalLen > dele.maxPackLen {
		pkt.err = errors.New(log.Sprintf("Body Len(%d) > MAXPACK_LEN(%d)", pkt.totalLen, dele.maxPackLen))
		goto Exit
	}

	if pkt.totalLen < DEFAULT_PACK_HEAD_LEN {
		pkt.err = errors.New(log.Sprintf("totalLen(%d) < DEFAULT_PACK_HEAD_LEN(%d)", pkt.totalLen, DEFAULT_PACK_HEAD_LEN))
		goto Exit
	}

	if pkt.err = (*client.GetConn()).SetWriteDeadline(time.Now().Add(dele.sendBlockTime)); pkt.err != nil {
		goto Exit
	}

	pkt.totalLen, pkt.err = client.GetConn().Write(pkt.buf)
	client.AddWriteLen(pkt.totalLen)
	if pkt.err == nil {
		return nil
	}

Exit:
	log.Debug("[SendMsg] Client(%s) Error: %v", client.Info(), pkt.err)
	return pkt.err
}

func (dele *DefaultNetDelegate) SendData(client ITcpClient, data []byte) error {
	pkt := struct {
		err      error
		dataLen  int
		writeLen int
	}{
		dataLen: len(data),
	}

	if pkt.dataLen > 0 {
		if pkt.dataLen > dele.maxPackLen {
			pkt.err = errors.New(log.Sprintf("Body Len(%d) > MAXPACK_LEN(%d)", pkt.dataLen, dele.maxPackLen))
			goto Exit
		}

		if pkt.err = (*client.GetConn()).SetWriteDeadline(time.Now().Add(dele.sendBlockTime)); pkt.err != nil {
			goto Exit
		}

		pkt.writeLen, pkt.err = client.GetConn().Write(data)
		if pkt.writeLen != pkt.dataLen {
			pkt.err = errors.New(log.Sprintf("writeLen(%d) != dataLen(%d)", pkt.writeLen, pkt.dataLen))
		}
		if pkt.err == nil {
			return nil
		}
	} else {
		pkt.err = errors.New("len(data) == 0")
	}

Exit:
	log.Debug("[SendData] Client(%s) Error: %v", client.Info(), pkt.err)
	//client.Stop()
	return pkt.err
}

func (dele *DefaultNetDelegate) HandleMsg(client ITcpClient, msg INetMsg) {
	defer util.HandlePanic()
	cmd := uint32(msg.MainCmd())<<16 | uint32(msg.SubCmd())
	if cb, ok := dele.handlerMap[cmd]; ok {
		if dele.Server != nil {
			dele.Server.Add(1)
			defer dele.Server.Done()
			if dele.Server.Running {
				cb(client, msg)
			}
		} else {
			cb(client, msg)
		}

		// if cb(msg) {
		// 	return
		// } else {
		// 	//msg.GetClient().Stop()
		// }
	} else {
		//log.Debug("%s %s HandleMsg Error: No Handler For MainCmd %d, SubCmd: %d", dele.tag, client.Info(), msg.MainCmd(), msg.SubCmd())
	}
}

func (dele *DefaultNetDelegate) Handle(mainCmd uint16, subCmd uint16, cb func(client ITcpClient, msg INetMsg)) {
	// dele.Lock()
	// defer dele.Unlock()
	cmd := uint32(mainCmd)<<16 | uint32(subCmd)
	if dele.handlerMap == nil {
		dele.handlerMap = make(map[uint32]func(client ITcpClient, msg INetMsg))
	}
	dele.handlerMap[cmd] = cb
}

// func (dele *DefaultNetDelegate) Unhandle(mainCmd uint16, subCmd uint16) {
// 	// dele.Lock()
// 	// defer dele.Unlock()
// 	cmd := uint32(mainCmd)<<16 | uint32(subCmd)
// 	delete(dele.handlerMap, cmd)
// }

func (dele *DefaultNetDelegate) GetRecvBlockTime() time.Duration {
	return dele.recvBlockTime
}
func (dele *DefaultNetDelegate) SetRecvBlockTime(recvBT time.Duration) {
	dele.Lock()
	defer dele.Unlock()
	dele.recvBlockTime = recvBT
}

func (dele *DefaultNetDelegate) GetSendBlockTime() time.Duration {
	return dele.sendBlockTime
}
func (dele *DefaultNetDelegate) SetSendBlockTime(sendBT time.Duration) {
	dele.Lock()
	defer dele.Unlock()
	dele.sendBlockTime = sendBT
}

func (dele *DefaultNetDelegate) GetRecvBufLen() int {
	return dele.recvBufLen
}
func (dele *DefaultNetDelegate) SetRecvBufLen(recvBL int) {
	dele.Lock()
	defer dele.Unlock()
	dele.recvBufLen = recvBL
}

func (dele *DefaultNetDelegate) GetSendBufLen() int {
	return dele.sendBufLen
}
func (dele *DefaultNetDelegate) SetSendBufLen(sendBL int) {
	dele.Lock()
	defer dele.Unlock()
	dele.sendBufLen = sendBL
}

func (dele *DefaultNetDelegate) GetSendQueueSize() int {
	return dele.sendQueueSize
}
func (dele *DefaultNetDelegate) SetSendQueueSize(sendBL int) {
	dele.Lock()
	defer dele.Unlock()
	dele.sendQueueSize = sendBL
}

func (dele *DefaultNetDelegate) GetMaxPackLen() int {
	return dele.maxPackLen
}
func (dele *DefaultNetDelegate) SetMaxPackLen(maxPL int) {
	dele.Lock()
	defer dele.Unlock()
	dele.maxPackLen = maxPL
}

func (dele *DefaultNetDelegate) IsNoDelay() bool {
	return dele.noDelay
}
func (dele *DefaultNetDelegate) SetNoDelay(nodelay bool) {
	dele.Lock()
	defer dele.Unlock()
	dele.noDelay = nodelay
}

func (dele *DefaultNetDelegate) IsKeepAlive() bool {
	return dele.keepAlive
}
func (dele *DefaultNetDelegate) SetKeepAlive(keppalive bool) {
	dele.Lock()
	defer dele.Unlock()
	dele.keepAlive = keppalive
}

func (dele *DefaultNetDelegate) GetAliveTime() time.Duration {
	return dele.aliveTime
}

func (dele *DefaultNetDelegate) SetAliveTime(aliveT time.Duration) {
	dele.Lock()
	defer dele.Unlock()
	dele.aliveTime = aliveT
}

func (dele *DefaultNetDelegate) GetTimer() *timer.Timer {
	return dele.asynctimer
}

func (dele *DefaultNetDelegate) SetTag(tag string) {
	dele.tag = tag
}
