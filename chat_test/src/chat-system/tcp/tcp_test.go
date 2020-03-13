package tcp

import (
	"fmt"
	"chat-system/coder"
	"testing"
	"time"
)

const (
	CMD_MAIN = 1
	CMD_ECHO = 1
)

func server_echo(client ITcpClient, msg INetMsg) {
	fmt.Printf("[server recv] [%s]: %s\n", client.Info(), string(msg.Body()))
	client.SendAsync(msg.Clone())
	fmt.Printf("[server send] [%s]: %s\n", client.Info(), string(msg.Body()))
}

func server() {
	server := NewTcpServer("tcp-server")
	delegate := server.GetDelegate()
	delegate.NewCoder(func() Coder { return coder.NewServerCoder() })
	delegate.Handle(CMD_MAIN, CMD_ECHO, server_echo)
	server.Serve("[::]:8888", time.Second*2)
}

func client_echo(client ITcpClient, msg INetMsg) {
	fmt.Printf("[client recv] [%s]: %s\n", client.Info(), string(msg.Body()))
}

func client() {
	clientdele := &DefaultNetDelegate{}
	clientdele.Handle(CMD_MAIN, CMD_ECHO, client_echo)

	started := false
	autoreconnect := true
	onconnected := func(c ITcpClient, ok bool) {
		if ok {
			if !started {
				started = true
				count := 0
				go func() {
					for {
						count++
						str := fmt.Sprintf("hello %d", count)
						msg := NewNetMsg(CMD_MAIN, CMD_ECHO, []byte(str))
						c.SendMsg(msg)
						fmt.Printf("[client send] [%s]: %s\n", c.Info(), string(msg.Body()))
						time.Sleep(time.Second)
					}
				}()
			}

		}
	}
	client := NewTcpClient(clientdele, "127.0.0.1:8888", 0, coder.NewClientCoder(), autoreconnect, onconnected)
	go client.Connect()

	//make(chan int) <- 0
}

func TestTcp(t *testing.T) {
	go func() {
		time.Sleep(time.Second)
		client()
	}()

	server()
}
