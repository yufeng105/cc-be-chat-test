package tcp

import (
	"encoding/binary"
	"chat-system/log"
	"chat-system/util"
)

type INetMsg interface {
	Clone() INetMsg

	Version() uint8
	SetVersion(version uint8)
	CheckCode() uint8
	SetCheckCode(code uint8)
	PackLen() uint16
	SetPackLen(length uint16)
	MainCmd() uint16
	SetMainCmd(cmd uint16)
	SubCmd() uint16
	SetSubCmd(cmd uint16)
	HeadLen() int
	TotalLen() int
	// TotalEncryptedLen() int
	Body() []byte
	BodyLen() int
	SetBody(data []byte)
	PackData() []byte
	// EncryptedPackData() []byte
	Encrypt(coder Coder) ([]byte, error)
	Decrypt(coder Coder) error
	//GetClient() ITcpClient
}

type NetMsg struct {
	//Client ITcpClient
	Buf []byte
	// Encrypted []byte
	// flag int32
}

func (msg *NetMsg) Clone() INetMsg {
	return &NetMsg{
		Buf: append([]byte{}, msg.Buf...),
	}
}

func (msg *NetMsg) Version() uint8 {
	return uint8(msg.Buf[0])
}

func (msg *NetMsg) SetVersion(version uint8) {
	msg.Buf[0] = byte(version)
}

func (msg *NetMsg) CheckCode() uint8 {
	return uint8(msg.Buf[1])
}

func (msg *NetMsg) SetCheckCode(code uint8) {
	msg.Buf[1] = byte(code)
}

func (msg *NetMsg) PackLen() uint16 {
	return uint16(binary.LittleEndian.Uint16(msg.Buf[2:]))
}

func (msg *NetMsg) SetPackLen(length uint16) {
	binary.LittleEndian.PutUint16(msg.Buf[2:], length)
}

func (msg *NetMsg) MainCmd() uint16 {
	return uint16(binary.LittleEndian.Uint16(msg.Buf[4:]))
}

func (msg *NetMsg) SetMainCmd(cmd uint16) {
	binary.LittleEndian.PutUint16(msg.Buf[4:], cmd)
}

func (msg *NetMsg) SubCmd() uint16 {
	return uint16(binary.LittleEndian.Uint16(msg.Buf[6:]))
}

func (msg *NetMsg) SetSubCmd(cmd uint16) {
	binary.LittleEndian.PutUint16(msg.Buf[6:], cmd)
}

func (msg *NetMsg) HeadLen() int {
	return DEFAULT_PACK_HEAD_LEN
}

func (msg *NetMsg) TotalLen() int {
	return len(msg.Buf)
}

// func (msg *NetMsg) TotalEncryptedLen() int {
// 	return len(msg.Encrypted)
// }

func (msg *NetMsg) Body() []byte {
	return msg.Buf[DEFAULT_PACK_HEAD_LEN:]
}

func (msg *NetMsg) BodyLen() int {
	return len(msg.Buf[DEFAULT_PACK_HEAD_LEN:])
}

func (msg *NetMsg) SetBody(data []byte) {
	needLen := len(data) - len(msg.Buf) + DEFAULT_PACK_HEAD_LEN
	if needLen > 0 {
		msg.Buf = append(msg.Buf, make([]byte, needLen)...)
	} else if needLen < 0 {
		msg.Buf = msg.Buf[:len(data)+DEFAULT_PACK_HEAD_LEN]
	}
	copy(msg.Buf[DEFAULT_PACK_HEAD_LEN:], data)
}

func (msg *NetMsg) PackData() []byte {
	return msg.Buf
}

// func (msg *NetMsg) EncryptedPackData() []byte {
// 	return msg.Encrypted
// }

func (msg *NetMsg) Encrypt(coder Coder) ([]byte, error) {
	// if atomic.CompareAndSwapInt32(&msg.flag, 0, 1) {
	if coder != nil {
		if encrypted, err := coder.Encrypt(append([]byte{}, msg.Buf...)); err == nil {
			return encrypted, err
		}
	}
	// }
	return msg.Buf, nil
}

func (msg *NetMsg) Decrypt(coder Coder) error {
	// if atomic.CompareAndSwapInt32(&msg.flag, 0, 1) {
	if coder != nil {
		decrypted, err := coder.Decrypt(msg.Buf)
		if err == nil {
			msg.Buf = decrypted
		}
		return err
	}
	// }
	return nil
}

// func (msg *NetMsg) GetClient() ITcpClient {
// 	return msg.Client
// }

func NewNetMsg(mainCmd uint16, subCmd uint16, data interface{}) *NetMsg {
	msg := NetMsg{}
	if data != nil {
		buf := util.DataToBuf(data)
		if buf != nil {
			if len(buf) <= DEFAULT_MAX_BODY_LEN {
				msg.Buf = append(make([]byte, DEFAULT_PACK_HEAD_LEN), buf...)
			} else {
				log.Debug("NewNetMsg failed: body len(%d) > DEFAULT_MAX_BODY_LEN(%d)", len(buf), DEFAULT_MAX_BODY_LEN)
			}
		}
	}
	if len(msg.Buf) == 0 {
		msg.Buf = make([]byte, DEFAULT_PACK_HEAD_LEN)
	}

	msg.Buf[0] = SOCKET_VER
	msg.SetPackLen(uint16(len(msg.Buf) + DEFAULT_PACK_HEAD_LEN))
	msg.SetMainCmd(mainCmd)
	msg.SetSubCmd(subCmd)
	return &msg
}
