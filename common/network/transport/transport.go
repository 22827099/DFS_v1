package transport

// Transport 定义网络传输接口
type Transport interface {
	// 启动传输服务
	Start() error
	// 停止传输服务
	Stop() error
	// 发送消息
	Send(destination string, data []byte) error
	// 设置接收消息的回调函数
	OnReceive(callback func(source string, data []byte))
}
