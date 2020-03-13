package main

import (
	"chat-system/badwords"
	"chat-system/coder"
	"chat-system/log"
	"chat-system/proto"
	"chat-system/tcp"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

var (
	tcpService = &TcpService{
		clients:  map[tcp.ITcpClient]*User{},
		nameVerb: map[string]*User{},
	}

	filterConfig = badwords.Init("list.txt")
)

type User struct {
	UserName  string
	LoginTime time.Time
}

type TcpService struct {
	sync.RWMutex
	clients  map[tcp.ITcpClient]*User
	nameVerb map[string]*User
}

/*********************** client management ***********************/
//client认证成功加入列表
func (s *TcpService) addClient(client tcp.ITcpClient, userName string) {
	s.Lock()
	defer s.Unlock()
	s.clients[client] = &User{UserName: userName, LoginTime: time.Now()}
	s.nameVerb[userName] = s.clients[client]
}

//client断开从列表移除
func (s *TcpService) deleteClient(client tcp.ITcpClient) {
	s.Lock()
	defer s.Unlock()
	user, have := s.clients[client]
	if have {
		delete(s.nameVerb, user.UserName)
	} else {
		log.Error("bug ^v", client.Ip())
	}
	delete(s.clients, client)
}

//广播消息给所有client
func (s *TcpService) broadcast(msgtype string, msg tcp.INetMsg) {
	s.RLock()
	defer s.RUnlock()

	for client, valid := range s.clients {
		if valid != nil {
			client.SendMsg(msg)
		}
	}
	log.Info("broadcast '%s', client num: %+v", msgtype, len(s.clients))
}

func (s *TcpService) getLoginTime(client tcp.ITcpClient) int64 {
	s.RLock()
	defer s.RUnlock()

	u, ok := s.clients[client]
	if ok {
		return time.Now().Unix() - u.LoginTime.Unix()
	}
	log.Error("bug:%v", client.Ip())
	return 0
}

/*********************** msg handlers ***********************/
//心跳包，原样回包
func (s *TcpService) onHeartBeat(client tcp.ITcpClient, msg tcp.INetMsg) {
	client.SendMsg(msg)
}

func (s *TcpService) onEcho(client tcp.ITcpClient, msg tcp.INetMsg) {
	log.Debug("[server recv] [%s]: %s\n", client.Info(), string(msg.Body()))
	client.SendAsync(msg.Clone())
	log.Debug("[server send] [%s]: %s\n", client.Info(), string(msg.Body()))
}

func (s *TcpService) onAuth(client tcp.ITcpClient, msg tcp.INetMsg) {
	req := &proto.AuthReq{}
	if err := json.Unmarshal(msg.Body(), req); err != nil {
		rsp := &proto.AuthRsp{
			ErrCode: -1,
		}
		rspmsg := tcp.NewNetMsg(CMD_MAIN, CMD_AUTH, rsp)
		client.SendAsync(rspmsg, func() {
			client.Stop()
		})
		log.Error("onAuth json.Unmarshal failed: %v", err)
		return
	}
	//简单判断是否为空，不判断重复
	if len(req.UserName) == 0 {
		rsp := &proto.AuthRsp{
			ErrCode: -1,
		}
		rspmsg := tcp.NewNetMsg(CMD_MAIN, CMD_AUTH, rsp)
		client.SendAsync(rspmsg, func() {
			client.Stop()
		})
		return
	}
	s.addClient(client, req.UserName)

	//登录成功返回历史记录
	history := historyRecord.getHistory()
	rsq := &proto.ChatRsp{}
	historyStr, err := json.Marshal(history)
	if err != nil {
		log.Error("send history err:%v", err)
		return
	}
	rsq.Msg = string(historyStr)
	log.Debug("historyStr:%v", rsq.Msg)
	client.SendMsg(tcp.NewNetMsg(CMD_MAIN, CMD_CHAT, rsq))
}

//如果是中文需要最语意处理，目前当作英文简单处理
func (s *TcpService) makePopular(msg string) {
	words := strings.Split(msg, " ")
	for _, v := range words {
		popularWordMgr.add(v)
	}
}

func (s *TcpService) onPopular(client tcp.ITcpClient, msg tcp.INetMsg) {
	log.Debug("onPopular:%v", client.Ip())
	popStrArray := popularWordMgr.get()
	if popStrArray == nil || len(popStrArray) == 0 {
		log.Debug("not found")
		return
	}
	popStrBytes, err := json.Marshal(popStrArray)
	if err != nil {
		log.Error("popStrArray json.Marshal err:%v,data:%v", err, popStrArray)
		return
	}
	popStr := string(popStrBytes)
	log.Info("popular words:%v", popStr)
	client.SendMsg(tcp.NewNetMsg(CMD_MAIN, CMD_CHAT, proto.ChatRsp{Msg: popStr}))
}

func (s *TcpService) onStats(client tcp.ITcpClient, msg string) {
	loginTime := s.getLoginTime(client)
	loginTimeStr := time.Unix(loginTime, 0).Format("02d 15h 04m 05s")
	log.Debug("onStats %v logintime:%v", msg, loginTimeStr)
	client.SendMsg(tcp.NewNetMsg(CMD_MAIN, CMD_CHAT, &proto.ChatRsp{Msg: loginTimeStr}))
}

func (s *TcpService) onChat(client tcp.ITcpClient, msg tcp.INetMsg) {
	req := &proto.ChatReq{}
	if err := json.Unmarshal(msg.Body(), req); err != nil {
		rsp := &proto.AuthRsp{
			ErrCode: -1,
		}
		rspmsg := tcp.NewNetMsg(CMD_MAIN, CMD_CHAT, rsp)
		client.SendAsync(rspmsg, func() {
			client.Stop()
		})
		log.Error("onChat json.Unmarshal failed: %v", err)
		return
	}

	if req.Msg == "/popular" {
		go s.onPopular(client, msg)
		return
	}

	msgLen := len(req.Msg)
	if msgLen > 7 {
		if strings.Index(req.Msg, "/stats ") == 0 {
			go s.onStats(client, req.Msg[6:msgLen])
			return
		}
	}

	txtSlice := strings.Split(req.Msg, "")
	badWords := badwords.Search(filterConfig, &txtSlice, "*")
	result := strings.Join(txtSlice, "")

	log.Debug("onChat:%v,txtSlice:%v,badWords:%v,result:%v", req.Msg, txtSlice, badWords, result)

	//没有过滤词才给加到历史记录
	if len(*badWords) == 0 {
		go func() {
			historyRecord.addHistory(req.Msg)
			s.makePopular(req.Msg)
		}()
	} else {
		req.Msg = result
	}

	log.Debug("onChat:%v", req.Msg)
	rspmsg := tcp.NewNetMsg(CMD_MAIN, CMD_CHAT, &proto.ChatReq{Msg: req.Msg})
	s.broadcast("chat-group", rspmsg)
}

func (s *TcpService) start() {
	server := tcp.NewTcpServer("service")
	delegate := server.GetDelegate()
	delegate.NewCoder(func() tcp.Coder { return coder.NewServerCoder() })
	delegate.Handle(CMD_MAIN, CMD_ECHO, tcpService.onEcho)
	delegate.Handle(CMD_MAIN, CMD_AUTH, tcpService.onAuth)
	delegate.Handle(CMD_MAIN, CMD_CHAT, tcpService.onChat)
	server.Serve("[::]:8888", time.Second*2)
}
