# 说明
使用 `github.com/gorilla/websocket` 与 `golang.org/x/crypto/ssh` 实现的 webssh，支持颜色以及自动补全

参考：https://github.com/myml/webssh ，在原项目的基础上新增 publickey 登陆、zmodem 上传下载（支持禁用 sz 或者 rz）


# 服务器端文档

## 快速开始
```go
...
import (
    ...

    "github.com/gorilla/websocket"
    "github.com/leffss/gowebssh"
)
...
id := r.Header.Get("Sec-WebSocket-Key")
webssh := gowebssh.NewWebSSH()
webssh.SetTerm(gowebssh.TermLinux)
webssh.SetBuffSize(8192)
webssh.SetId(id)
webssh.SetConnTimeOut(15 * time.Second)
webssh.DisableSZ()
//webssh.DisableRZ()
...
upGrader := websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
    //Subprotocols: []string{r.Header.Get("Sec-WebSocket-Protocol")},
    Subprotocols: []string{"webssh"},
    ReadBufferSize: 8192,
    WriteBufferSize: 8192,
}
ws, _ := upGrader.Upgrade(w, r, nil)
webssh.AddWebsocket(ws)
```

# 客户端文档

## 消息类型

```go
type messageType string

const (
	messageTypeAddr      = "addr"
	messageTypeTerm      = "term"
	messageTypeLogin     = "login"
	messageTypePassword  = "password"
	messageTypePublickey = "publickey"
	messageTypeStdin     = "stdin"
	messageTypeStdout    = "stdout"
	messageTypeStderr    = "stderr"
	messageTypeResize    = "resize"
	messageTypeIgnore    = "ignore"
	messageTypeConsole   = "console"
)

type message struct {
	Type messageType  `json:"type"`
	Data []byte       `json:"data,omitempty"`
	Cols int          `json:"cols,omitempty"`
	Rows int          `json:"rows,omitempty"`
}
```

## 消息协议

1. 地址 `{type:"addr",data:"$addr"}`  
   地址格式： ip:port，例如 192.168.223.111:22
2. 登录 `{type:"login",data:"$username"}`
3. 设置 term 终端类型 `{type:"term",data:"$term"}`    # 可不设置，默认 xterm
4. 验证 `{type:"password",data:"$password"}` or `{type:"publickey",data:"$publickey"}`
5. 窗口大小调整 `{type:"resize",cols:40,rows:80}`
6. 忽略数据流 `{type:"ignore",data:"$data"}`     # 客户端发送到服务端，服务器忽略，可以用于 zmodem 文件传输记录
7. console 数据流 `{type:"console",data:"$data"}`     # 服务端发送到客户端，客户端显示到 console 控制台的数据，
可以用于 zmodem 文件传输时的 debug 信息
8. alert 数据流 `{type:"alert",data:"$data"}`  # # 服务端发送到客户端的 alert 信息
9. 标准流数据  
   `{type:"stdin",data:"$data"}`
   `{type:"stdout",data:"$data"}`
   `{type:"stderr",data:"$data"}`  
   客户端发送 stdin, 接收 stdout, stderr

## Data 数据

消息的 data 数据使用 base64 编码传输，JavaScript 的`atob & btoa`可用于 base64 编码，但对 utf8 有兼容性问题，
要使用`decodeURIComponent & encodeURIComponent`做包裹, 以下是实现

```javascript
function utf8_to_b64(rawString) {
  return btoa(unescape(encodeURIComponent(rawString)));
}

function b64_to_utf8(encodeString) {
  return decodeURIComponent(escape(atob(encodeString)));
}
```

# 实例

具体实例参考 `example` 文件夹

# 预览
![效果](https://github.com/leffss/gowebssh/blob/master/screenshots/1.PNG?raw=true)
![效果](https://github.com/leffss/gowebssh/blob/master/screenshots/2.PNG?raw=true)
![效果](https://github.com/leffss/gowebssh/blob/master/screenshots/3.PNG?raw=true)
![效果](https://github.com/leffss/gowebssh/blob/master/screenshots/4.PNG?raw=true)
![效果](https://github.com/leffss/gowebssh/blob/master/screenshots/5.PNG?raw=true)
![效果](https://github.com/leffss/gowebssh/blob/master/screenshots/6.PNG?raw=true)
