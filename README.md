# 说明
使用 `github.com/gorilla/websocket` 与 `golang.org/x/crypto/ssh` 实现的 webssh，支持颜色以及自动补全

参考：https://github.com/myml/webssh 实现，原项目只实现了 password 登陆，新增 publickey 登陆

# 服务器端文档

## 快速开始

使用相同 ID,分别添加 net.conn,websocket.conn

```go
...
import (
    ...

    "github.com/gorilla/websocket"
    "github.com/leffss/gowebssh"
)
...
webssh := gowebssh.NewWebSSH()
webssh.SetTerm("linux")
webssh.SetBuffSize(1024)
...
conn, _ := net.Dial("tcp", "127.0.0.1:22")
webssh.AddSSHConn("$uuid", conn)
...
upGrader := websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
    Subprotocols: []string{r.Header.Get("Sec-WebSocket-Protocol")},
    ReadBufferSize: 1024,
    WriteBufferSize: 1024,
}
ws, _ := upGrader.Upgrade(w, r, nil)
webssh.AddWebsocket("$uuid", ws)
```

# 客户端文档

## 消息类型

```go
type messageType string
const (
	messageTypeStdin     = "stdin"
	messageTypeStdout    = "stdout"
	messageTypeStderr    = "stderr"
	messageTypeResize    = "resize"
	messageTypeLogin     = "login"
	messageTypePassword  = "password"
	messageTypePublickey = "publickey"
)
type message struct {
	Type messageType `json:"type"`
	Data []byte      `json:"data"`
	Cols int         `json:"cols"`
	Rows int         `json:"rows"`
}
```

## 消息协议

1. 登录 `{type:"login",data:"$username"}`
2. 验证 `{type:"password",data:"$password"}` or `{type:"publickey",data:"$publickey"}`
3. 窗口大小调整 `{type:"resize",cols:40,rows:80}`
4. 标准流数据  
   `{type:"stdin",data:"$data"}`
   `{type:"stdout",data:"$data"}`
   `{type:"stderr",data:"$data"}`  
   客户端发送 stdin,接收 stdout,stderr

## Data 数据

消息的 data 数据使用 base64 编码传输，JavaScript 的`atob & btoa`可用于 base64 编码，但对 utf8 有兼容性问题，要使用`decodeURIComponent & encodeURIComponent`做包裹,以下是实现

```javascript
function atou(encodeString) {
  return decodeURIComponent(escape(atob(encodeString)));
}
function utoa(rawString) {
  return btoa(encodeURIComponent(rawString));
}
```

## 实例

具体实例参考 `example` 文件夹
