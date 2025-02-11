# gowebssh

> 中文文档 [README](./README.md)



Webssh implemented by `github.com/gorilla/websocket` and `golang.org/x/crypto/ssh`, which supports ansi color and  `tab` button for automatic completion command.

Reference resources: https://github.com/myml/webssh , on the basis of the original project, add public key login, zmodem upload and download file(support disabling sz or rz)



# Server

## quick start
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



# Client

## message type

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



## message

1. address `{type:"addr",data:"$addr"}`  
   format： `ip:port` ，like `192.168.223.111:22`
2. login `{type:"login",data:"$username"}`
3. set term type `{type:"term",data:"$term"}`    # default xterm
4. auth `{type:"password",data:"$password"}` or `{type:"publickey",data:"$publickey",passphrase:"$passphrase"}`
5. resize window `{type:"resize",cols:40,rows:80}`
6. ignore data stream `{type:"ignore",data:"$data"}`     # client send to server，server  will ignore，could use for zmodem file transmission record
7. console stream `{type:"console",data:"$data"}`     # server send to client, client display it on the web browser console, could use for debug zmodem file transmission
8. alert stream `{type:"alert",data:"$data"}`   # server send to client, alert message
9. standard stream data  
   `{type:"stdin",data:"$data"}`
   `{type:"stdout",data:"$data"}`
   `{type:"stderr",data:"$data"}`  
   client send stdin data, receive stdout, stderr data



## Data transmission

use base64 encoding during transmission

```javascript
function utf8_to_b64(rawString) {
  return btoa(unescape(encodeURIComponent(rawString)));
}

function b64_to_utf8(encodeString) {
  return decodeURIComponent(escape(atob(encodeString)));
}
```



# Example

check in `example` path



# Preview

![效果](https://github.com/leffss/gowebssh/blob/master/screenshots/1.PNG?raw=true)
![效果](https://github.com/leffss/gowebssh/blob/master/screenshots/2.PNG?raw=true)
![效果](https://github.com/leffss/gowebssh/blob/master/screenshots/3.PNG?raw=true)
![效果](https://github.com/leffss/gowebssh/blob/master/screenshots/4.PNG?raw=true)
![效果](https://github.com/leffss/gowebssh/blob/master/screenshots/5.PNG?raw=true)
![效果](https://github.com/leffss/gowebssh/blob/master/screenshots/6.PNG?raw=true)
