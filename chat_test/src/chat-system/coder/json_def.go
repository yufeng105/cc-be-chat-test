package coder

type VerifyToekn struct {
	UserID int    //用户ID
	Token  string //token
}

type VerifyTokenSuccess struct {
	ChatLength     int
	LimitChatTimes int
}

type PUSH_ServerInfo struct {
	ServerID  int
	OnLineNum int
}

type ErrJSON struct {
	ErrCode int
	ErrMsg  string
}
