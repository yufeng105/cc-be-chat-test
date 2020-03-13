package proto

const MAX_BODY_LEN = 16384 - 8



type AuthReq struct {
	UserName string `json:"username"` //username
	//Token     string `json:"token"`     //token
	//Timestamp int64  `json:"timestamp"` //时间戳
	//Sign      string `json:"sign"`      //hmac sign: UserID+Token+Timestamp, key=Account
}

type AuthRsp struct {
	ErrCode int `json:"code"` //错误码，0为成功，非0失败
}

type ChatReq struct {
	Msg string `json:"msg"` //用户ID
}

type ChatRsp struct {
	ErrCode int    `json:"code"` //错误码，0为成功，非0失败
	Msg     string `json:"msg"`  //error
}
