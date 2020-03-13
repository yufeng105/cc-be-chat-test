package main

import (
	"chat-system/coder"
	"chat-system/log"
	"chat-system/proto"
	"chat-system/tcp"
	"encoding/json"
	"time"
)

const (
	CMD_MAIN = 1

	CMD_ECHO = 1
	CMD_AUTH = 2
	CMD_CHAT = 3
)

func client_echo(client tcp.ITcpClient, msg tcp.INetMsg) {
	log.Debug("[client recv] [%s]: %s\n", client.Info(), string(msg.Body()))
}

func client() {
	clientdele := &tcp.DefaultNetDelegate{}
	clientdele.Handle(CMD_MAIN, CMD_ECHO, client_echo)

	started := false
	autoreconnect := true
	onconnected := func(c tcp.ITcpClient, ok bool) {
		if ok {
			if !started {
				started = true
				testCount := 0
				cmd := CMD_AUTH
				go func() {
					for {
						switch cmd {
						case CMD_AUTH:
							req := proto.AuthReq{UserName: "test1111"}
							str, _ := json.Marshal(req)
							msg := tcp.NewNetMsg(CMD_MAIN, CMD_AUTH, []byte(str))
							c.SendMsg(msg)
						case CMD_CHAT:
							msgStr := ""
							if testCount == 0 {
								msgStr = "1111111111"
								testCount = 1
							} else if testCount == 1 {
								msgStr = "/popular"
								testCount = 2
							} else if testCount == 2 {
								msgStr = "/stats  test1111"
								testCount = 3
							} else {
								msgStr = "12344"
							}
							req := proto.ChatReq{Msg: msgStr}
							str, _ := json.Marshal(req)
							msg := tcp.NewNetMsg(CMD_MAIN, CMD_CHAT, []byte(str))
							c.SendMsg(msg)
						}
						time.Sleep(time.Second)
						if cmd == CMD_AUTH {
							cmd = CMD_CHAT
						}
					}
				}()
			}

		}
	}
	client := tcp.NewTcpClient(clientdele, "127.0.0.1:8888", 0, coder.NewClientCoder(), autoreconnect, onconnected)
	go client.Connect()

	//make(chan int) <- 0
}

func test_setup() {
	go func() {
		time.Sleep(time.Second)
		client()
	}()

	tcpService.start()
}

func setup() {
	tcpService.start()
}

//
func main() {
	//test_setup()
	setup()
}
