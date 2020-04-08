package protocol

import (
	"bytes"
	"encoding/json"
	"github.com/tebeka/selenium"
)

//k : paymentid, v:paymentid 这么创建便于发送消息，知道哪个webpage退出
type FFChAppTag struct {
	ChAppUrl   chan string //二维码的通道
	ChQuit     chan string    //主线程收到退出消息的通道
	ChSelfExit chan string    //自己退出的消息通道
	WebPage    selenium.WebDriver
}

// push 接口请求协议格式
type PushReqTag struct {
	OrderNo string `json:"orderno"`
	Account string `json:"account"`
	Amount string `json:"amount"`
	ChanelCode string `json:"chanelcode"`
	ChanelType string `json:"chaneltype"`
	OrderTime string `json:"ordertime"`
	CallbackUrl string `json:"callbackurl"`
	PayType string `json:"paytype"`
	Sign string `json:"sign"`
}

// 成功返回结构
type PushResSuccessTag struct {
	OrderNo string `json:"orderno"`
	AppPayUrl string `json:"apppayurl"`
	Status string `json:"status"`
	Msg string `json:"msg"`
	Sign string `json:"sign"`
}

// http错误码
type HttpErrno struct {
	OrderNo string `json:"orderno"`
	Status string `json:"status"`
	Msg string `json:"msg"`
}

//序列化
func MakeJson(obj interface{}) (string, error) {
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	err := jsonEncoder.Encode(obj)
	if err != nil {
		return "", err
	}
	return bf.String(), nil
}