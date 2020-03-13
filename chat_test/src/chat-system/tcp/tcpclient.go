package tcp

import (
	"chat-system/log"
	"chat-system/timer"
	"chat-system/util"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	AUTO_RECONN_TAG = "autoreconnect"
)

var (
	asynctimer = timer.NewTimer()
)

type Coder interface {
	Encode(uint16, uint16, interface{}) ([]byte, error)
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
}

type ITcpClient interface {
	Lock()
	Unlock()
	Ip() string
	RealIp() string
	Info() string
	Index() int
	Stop()
	StopAsync()
	Connect()
	Start() bool
	GetConn() *net.TCPConn
	GetDelegate() ITNetDelegate
	SendMsg(msg INetMsg) error
	SendData(data []byte) error
	SendAsync(msg interface{}, argv ...interface{}) error
	SetDelegate(dele ITNetDelegate)
	AddCloseCB(key interface{}, cb func(client ITcpClient))
	RemoveCloseCB(key interface{})
	Coder() Coder
	SetCoder(coder Coder)
	SetRealIP(ip string)
	AddReadLen(int)
	ReadLen() int
	AddWriteLen(int)
	WriteLen() int
}

type TcpClient struct {
	sync.Mutex
	Conn            *net.TCPConn
	Delegate        ITNetDelegate
	Idx             int
	Addr            string
	chSend          chan *AsyncMsg
	closeCB         map[interface{}]func(client ITcpClient)
	Valid           bool
	Running         bool
	EnableReconnect bool
	onConnected     func(ITcpClient, bool)
	UserData        interface{}
	coder           Coder
	realip          string
	readLen         int
	writeLen        int
}

type AsyncMsg struct {
	//msg  INetMsg
	v  interface{}
	cb func()
}

func (client *TcpClient) AddReadLen(l int) {
	client.readLen += l
}

func (client *TcpClient) ReadLen() int {
	return client.readLen
}

func (client *TcpClient) AddWriteLen(l int) {
	client.writeLen += l
}

func (client *TcpClient) WriteLen() int {
	return client.writeLen
}

func (client *TcpClient) Ip() string {
	// client.Lock()
	// defer client.Unlock()
	if client.realip != "" {
		return client.realip
	}
	if client.GetConn() != nil {
		return strings.Split(client.GetConn().RemoteAddr().String(), ":")[0]
	}
	return "0.0.0.0"
}

func (client *TcpClient) RealIp() string {
	if client.realip != "" {
		return client.realip
	}
	return "0.0.0.0"
}

func (client *TcpClient) Info() string {
	if client.GetConn() != nil {
		return client.Ip() + ":" + strings.Split(client.GetConn().RemoteAddr().String(), ":")[1] + fmt.Sprintf(" (in:%v,out:%v)", client.readLen, client.writeLen)
	} else {
		return "-"
	}

	// return fmt.Sprintf("Client(Idx: %v, Addr: %s <-> %s)", client.Idx, addr, client.Addr)
	// return fmt.Sprintf("Client(Idx: %v, Addr: %s <-> %s)", client.Idx, addr, client.Addr)
}

func (client *TcpClient) Index() int {
	return client.Idx
}

func (client *TcpClient) Coder() Coder {
	return client.coder
}

func (client *TcpClient) SetCoder(coder Coder) {
	client.coder = coder
}

func (client *TcpClient) SetRealIP(ip string) {
	client.Lock()
	defer client.Unlock()
	client.realip = ip
}

func (client *TcpClient) GetConn() *net.TCPConn {
	return client.Conn
}

func (client *TcpClient) GetDelegate() ITNetDelegate {
	return client.Delegate
}

func (client *TcpClient) AddCloseCB(key interface{}, cb func(client ITcpClient)) {
	client.Lock()
	defer client.Unlock()
	if client.Running {
		client.closeCB[key] = cb
	}
}

func (client *TcpClient) RemoveCloseCB(key interface{}) {
	client.Lock()
	defer client.Unlock()
	//if client.Running {
	delete(client.closeCB, key)
	//}
}

func (client *TcpClient) Stop() {
	defer util.HandlePanic()
	client.Lock()
	running := client.Running
	client.Running = false
	client.Unlock()

	if running {
		if client.Conn != nil {
			client.Conn.Close()
		}

		if !client.EnableReconnect {
			if client.chSend != nil {
				close(client.chSend)
				client.chSend = nil
			}
			if len(client.closeCB) > 0 {
				for _, cb := range client.closeCB {
					cb(client)
				}
			}
			client.Delegate.OnDisconnected(client)
		} else {
			util.Go(func() {
				time.Sleep(time.Second)
				client.Connect()
			})
		}
	}
}

func (client *TcpClient) StopAsync() {
	t := client.Delegate.GetTimer()
	if t == nil {
		t = asynctimer
	}
	//go func() {
	t.Once(1, func() {
		defer util.HandlePanic()
		client.Lock()
		running := client.Running
		client.Running = false
		client.Unlock()

		if running {

			if client.Conn != nil {
				client.Conn.Close()
			}

			if !client.EnableReconnect {
				if client.chSend != nil {
					close(client.chSend)
					client.chSend = nil
				}
				if len(client.closeCB) > 0 {
					for _, cb := range client.closeCB {
						cb(client)
					}
				}
			} else {
				util.Go(func() {
					time.Sleep(time.Second)
					client.Connect()
				})
			}
		}
	})
	//}()
}

func (client *TcpClient) writer() {
	pkt := struct {
		asyncMsg *AsyncMsg
		ok       bool
		msg      INetMsg
		data     []byte
		chsend   chan *AsyncMsg
	}{}

	send := func() {
		client.Lock()
		defer client.Unlock()
		defer util.HandlePanic()
		if pkt.msg, pkt.ok = pkt.asyncMsg.v.(INetMsg); pkt.ok {
			client.Delegate.SendMsg(client, pkt.msg)
		} else if pkt.data, pkt.ok = pkt.asyncMsg.v.([]byte); pkt.ok {
			client.Delegate.SendData(client, pkt.data)
		} else {
			log.Error("Send AsyncMsg Error: Invalid Msg: %v")
		}
	}

	// if client.chSend == nil {
	// 	client.chSend = make(chan *AsyncMsg, client.Delegate.GetSendQueueSize())
	// }
	if pkt.chsend = client.chSend; pkt.chsend != nil {
		for {
			if pkt.asyncMsg, pkt.ok = <-pkt.chsend; pkt.ok {
				send()
				if pkt.asyncMsg.cb != nil {
					util.Safe(pkt.asyncMsg.cb)
				}
			} else {
				return
			}
		}
	}
}

func (client *TcpClient) SendMsg(msg INetMsg) error {
	// client.Lock()
	// defer func() {
	// 	client.Unlock()
	// 	recover()
	// }()
	// if client.Running {
	// 	return client.Delegate.SendMsg(client, msg)
	// }
	// return errors.New(fmt.Sprintf("Client(%s) is not running", client.Info()))
	return client.SendAsync(msg)
}

func (client *TcpClient) SendData(data []byte) error {
	// client.Lock()
	// defer func() {
	// 	client.Unlock()
	// 	recover()
	// }()
	// if client.Running {
	// 	return client.Delegate.SendData(client, data)
	// }
	// return errors.New(fmt.Sprintf("Client(%s) is not running", client.Info()))
	return client.SendAsync(data)
}

func (client *TcpClient) SendAsync(msg interface{}, argv ...interface{}) error {
	client.Lock()
	defer client.Unlock()
	if client.Running {
		asyncmsg := &AsyncMsg{
			v:  msg,
			cb: nil,
		}
		if len(argv) > 0 {
			/*if cb, ok := (argv[0]).(func()); ok {
				asyncmsg.cb = cb
			}*/
			asyncmsg.cb, _ = argv[0].(func())
		}
		if client.chSend != nil {
			select {
			case client.chSend <- asyncmsg:
				break
				// case <-time.After(client.Delegate.GetSendBlockTime()):
				// 	log.Debug("%s SendAsync Timeout, Msg: %v", client.Info(), msg)
				// 	return errors.New("timeout")
			default:
				log.Debug("%s SendAsync Failed: Buffer Is Full, Msg: %v", client.Info(), msg)
				return errors.New("timeout")
			}
		}
	} else {
		if len(argv) > 0 {
			/*if cb, ok := (argv[0]).(func()); ok {
				asyncmsg.cb = cb
			}*/
			if cb, ok := argv[0].(func()); ok {
				cb()
			}
		}
	}

	return nil
}

func (client *TcpClient) reader() {
	for {
		imsg := client.Delegate.RecvMsg(client)
		if imsg == nil {
			goto Exit
		}
		client.Delegate.HandleMsg(client, imsg)
	}

Exit:
	client.Stop()

}

func (client *TcpClient) Start() bool {
	if err := client.GetConn().SetKeepAlive(true); err != nil {
		log.Error("%s SetKeepAlive Err: %v.", client.Info())
		goto ErrExit
	}

	if err := client.GetConn().SetKeepAlivePeriod(client.Delegate.GetAliveTime()); err != nil {
		log.Error("%s SetKeepAlivePeriod Err: %v.", client.Info(), err)
		goto ErrExit
	}

	if err := (*client.GetConn()).SetReadBuffer(client.Delegate.GetRecvBufLen()); err != nil {
		log.Error("%s SetReadBuffer Err: %v.", client.Info(), err)
		goto ErrExit
	}
	if err := (*client.GetConn()).SetWriteBuffer(client.Delegate.GetSendBufLen()); err != nil {
		log.Error("%s SetWriteBuffer Err: %v.", client.Info(), err)
		goto ErrExit
	}
	if err := (*client.GetConn()).SetNoDelay(client.Delegate.IsNoDelay()); err != nil {
		log.Error("%s SetNoDelay Err: %v.", client.Info(), err)
		goto ErrExit
	}

	util.Go(client.reader)
	util.Go(client.writer)

	return true

ErrExit:
	client.GetConn().Close()
	return false
}

func (client *TcpClient) SetDelegate(dele ITNetDelegate) {
	client.Delegate = dele
}

func (client *TcpClient) Connect() {
	client.Lock()
	defer client.Unlock()
	tcpAddr, err := net.ResolveTCPAddr("tcp", client.Addr)
	if err != nil {
		log.Debug("TcpClient Connect ResolveTCPAddr Failed, err: %s, Addr: %s", err, client.Addr)
		return
	}
	//Dial:
	client.Conn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		//log.Debug("TcpClient Connect DialTCP(%s) Failed, err: %s", client.Addr, err)
		goto ErrExit
	}
	if !client.Start() {
		goto ErrExit
	}

	client.Running = true
	// if client.EnableReconnect {
	// 	t := client.Delegate.GetTimer()
	// 	if t == nil {
	// 		t = asynctimer
	// 	}
	// 	t.Once(1, func() {
	// 		client.AddCloseCB(AUTO_RECONN_TAG, func(c ITcpClient) {
	// 			util.Async(func() {
	// 				time.Sleep(time.Second)
	// 				c.Connect()
	// 			})
	// 		})
	// 	})
	// }
	if client.onConnected != nil {
		util.Go(func() {
			client.onConnected(client, true)
		})
	}
	return
ErrExit:
	if client.EnableReconnect {
		/*util.Go(func() {
			time.Sleep(time.Second)
			client.Connect()
		})*/
		util.Go(func() {
			time.Sleep(time.Second)
			client.Connect()
		})
		//goto Dial
	} else {
		if client.onConnected != nil {
			util.Go(func() {
				client.onConnected(client, false)
			})
		}
		client.StopAsync()
	}
}

func newTcpClient(dele ITNetDelegate, conn *net.TCPConn, idx int, coder Coder) *TcpClient {
	client := &TcpClient{
		Conn:            conn,
		Delegate:        dele,
		Idx:             idx,
		closeCB:         map[interface{}]func(client ITcpClient){},
		chSend:          make(chan *AsyncMsg, dele.GetSendQueueSize()),
		Valid:           false,
		Running:         true,
		EnableReconnect: false,
		onConnected:     nil,
		coder:           coder,
	}

	if conn != nil {
		client.Addr = conn.RemoteAddr().String()
	}
	if runtime.GOOS != "windows" && conn != nil {
		file, _ := conn.File()
		client.Idx = int(file.Fd())
	}

	return client
}

func NewTcpClient(dele ITNetDelegate, serveraddr string, idx int, coder Coder, reconn bool, onconnected func(ITcpClient, bool)) ITcpClient {
	dele.Init()
	client := newTcpClient(dele, nil, idx, coder)
	client.Addr = serveraddr
	client.EnableReconnect = reconn
	client.onConnected = onconnected
	// if reconn {
	// 	client.AddCloseCB(AUTO_RECONN_TAG, func(c ITcpClient) {
	// 		util.Go(func() {
	// 			time.Sleep(time.Second)
	// 			c.Connect()
	// 		})
	// 	})
	// }
	return client
}

func Ping(addr string) bool {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return false
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	defer conn.Close()
	if err != nil {
		return false
	}

	return true
}
