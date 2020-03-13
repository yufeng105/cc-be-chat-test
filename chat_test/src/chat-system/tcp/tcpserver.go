package tcp

import (
	"chat-system/log"
	"chat-system/util"
	"net"
	"os"
	//"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	servers      = make(map[string]*TcpServer)
	serversMutex = sync.Mutex{}

	_client_rm_from_server = "_*&^..3!_"
)

type TcpServer struct {
	sync.RWMutex
	sync.WaitGroup
	Name        string
	Running     bool
	clientNum   int
	currLoad    int32
	maxLoad     int32
	stopTimeout time.Duration
	listener    *net.TCPListener
	delegate    ITNetDelegate
	clients     map[ITcpClient]struct{}
}

func (server *TcpServer) addClient(client ITcpClient) {
	server.Lock()
	defer server.Unlock()
	server.clients[client] = struct{}{}
	atomic.AddInt32(&server.currLoad, 1)
}

func (server *TcpServer) deleClient(client ITcpClient) {
	server.Lock()
	defer server.Unlock()
	delete(server.clients, client)
	atomic.AddInt32(&server.currLoad, -1)
}

func (server *TcpServer) stopClients() {
	server.Lock()
	defer server.Unlock()

	for client, _ := range server.clients {
		client.RemoveCloseCB(_client_rm_from_server)
		client.Stop()
	}

	server.clients = map[ITcpClient]struct{}{}
}

func (server *TcpServer) startListener(addr string) {
	defer log.Debug("[TcpServer %s] Stopped.", server.Name)
	var (
		idx       int
		err       error
		conn      *net.TCPConn
		tcpAddr   *net.TCPAddr
		client    ITcpClient
		currload  int32
		tempDelay time.Duration
	)

	tcpAddr, err = net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Debug("[TcpServer %s] ResolveTCPAddr error: %v", server.Name, err)
		return
	}

	server.listener, err = net.ListenTCP("tcp", tcpAddr)

	if err != nil {
		log.Debug("[TcpServer %s] Listening error: %v", server.Name, err)
		return
	}

	defer server.listener.Close()

	server.Running = true

	log.Debug("[TcpServer %s] Running on: \"%s\"", server.Name, tcpAddr.String())
	for server.Running {
		if conn, err = server.listener.AcceptTCP(); err == nil {
			if server.delegate != nil {
				server.clientNum++
				// if runtime.GOOS == "linux" {
				// 	if file, err := conn.File(); err == nil {
				// 		idx = int(file.Fd())
				// 	}
				// } else {
				// 	idx = server.clientNum
				// }
				idx = server.clientNum
				currload = atomic.LoadInt32(&server.currLoad)
				if server.maxLoad <= 0 || currload < server.maxLoad {
					client = server.delegate.NewTcpClient(conn, idx)
					if client.Start() {
						server.addClient(client)
						client.AddCloseCB(_client_rm_from_server, server.deleClient)
						server.delegate.OnConnect(client)
					} else {
						conn.Close()
					}
				} else {
					conn.Close()
				}
			}
		} else {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Debug("[TcpServer %s] Accept error: %v; retrying in %v", server.Name, err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}

			log.Debug("[TcpServer %s] Accept error: %v", server.Name, err)
			break
		}
	}
}

func (server *TcpServer) Start(addr string) {
	server.Lock()
	running := server.Running
	if !server.Running {
		server.Running = true
	}
	server.Unlock()

	if !running {
		server.Add(1)
		defer deleteTcpServer(server.Name)
		server.startListener(addr)
	} else {

	}
}

func (server *TcpServer) Stop() {
	defer util.HandlePanic()

	server.Lock()
	running := server.Running
	server.Running = false
	server.Unlock()

	if !running {
		return
	}

	server.listener.Close()

	server.Done()

	if server.stopTimeout > 0 {
		time.AfterFunc(server.stopTimeout, func() {
			log.Debug("[TcpServer %s] Stop Timeout.", server.Name)
			time.Sleep(time.Second / 10)
			os.Exit(-1)
		})
	}

	log.Debug("[TcpServer %s] Stop Waiting...", server.Name)
	server.Wait()

	server.stopClients()

	if server.delegate != nil {
		server.delegate.OnServerStop()
	}

	time.Sleep(time.Second / 10)
	log.Debug("[TcpServer %s] Stop Done.", server.Name)
}

func (server *TcpServer) StopWithTimeout(to time.Duration) {
	server.stopTimeout = to
	server.Stop()
}

func (server *TcpServer) Serve(addr string, args ...interface{}) {
	util.Go(func() {
		server.Start(addr)
	})

	if len(args) > 0 {
		if to, ok := args[0].(time.Duration); ok {
			server.stopTimeout = to
		}
	}

	util.HandleSignal(func(sig os.Signal) {
		if sig == syscall.SIGTERM || sig == syscall.SIGINT {
			server.Stop()
			os.Exit(0)
		}
	})

}

func (server *TcpServer) GetDelegate() ITNetDelegate {
	server.Lock()
	defer server.Unlock()
	return server.delegate
}

func (server *TcpServer) SetDelegate(delegate ITNetDelegate) {
	server.Lock()
	defer server.Unlock()
	delegate.Init()
	server.delegate = delegate
	delegate.SetServer(server)
}

func (server *TcpServer) CurrLoad() int32 {
	return atomic.LoadInt32(&server.currLoad)
}

func (server *TcpServer) MaxLoad() int32 {
	return server.maxLoad
}

func (server *TcpServer) SetMaxLoad(maxLoad int32) {
	server.maxLoad = maxLoad
}

func NewTcpServer(name string) *TcpServer {
	serversMutex.Lock()
	defer serversMutex.Unlock()

	if _, ok := servers[name]; ok {
		log.Debug("NewTcpServer Error: (TcpServer-%s) already exists.", name)
		return nil
	}

	server := &TcpServer{
		Name:      name,
		Running:   false,
		clientNum: 0,
		currLoad:  0,
		maxLoad:   0,
		listener:  nil,
		clients:   map[ITcpClient]struct{}{},
	}

	server.SetDelegate(&DefaultNetDelegate{})

	servers[name] = server

	return server
}

func deleteTcpServer(name string) {
	serversMutex.Lock()
	defer serversMutex.Unlock()

	delete(servers, name)
}

func GetTcpServerByName(name string) (*TcpServer, bool) {
	serversMutex.Lock()
	defer serversMutex.Unlock()

	server, ok := servers[name]
	if !ok {
		log.Debug("GetTcpServerByName Error: TcpServer %s doesn't exists.", name)
	}
	return server, ok
}
