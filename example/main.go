package main

import (
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/leffss/gowebssh"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("./html")))
	http.HandleFunc("/api/ssh", func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("Sec-WebSocket-Key")
		addr := r.URL.Query().Get("addr")
		var webssh = gowebssh.NewWebSSH()
		// term 可以使用 ansi, linux, vt100, xterm, dumb，除了 dumb外其他都有颜色显示, 默认 xterm
		webssh.SetTerm("linux")
		webssh.SetBuffSize(1024)

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Panic(err)
		}
		webssh.AddSSHConn(id, conn)

		upGrader := websocket.Upgrader{
			// cross origin domain
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			// 处理 Sec-WebSocket-Protocol Header
			Subprotocols: []string{r.Header.Get("Sec-WebSocket-Protocol")},
			ReadBufferSize: 1024,
			WriteBufferSize: 1024,
		}

		ws, err := upGrader.Upgrade(w, r, nil)

		if err != nil {
			log.Panic(err)
		}

		webssh.AddWebsocket(id, ws)
	})

	log.Println("start")
	http.ListenAndServe(":8000", nil)
}
