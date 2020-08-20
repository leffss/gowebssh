package gowebssh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"net/url"
	"time"
)

var (
	// sz fmt.Sprintf("%+q", "rz\r**\x18B00000000000000\r\x8a\x11")
	//ZModemSZStart = []byte{13, 42, 42, 24, 66, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 13, 138, 17}
	ZModemSZStart = []byte{42, 42, 24, 66, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 13, 138, 17}
	// sz 结束 fmt.Sprintf("%+q", "\r**\x18B0800000000022d\r\x8a")
	//ZModemSZEnd = []byte{13, 42, 42, 24, 66, 48, 56, 48, 48, 48, 48, 48, 48, 48, 48, 48, 50, 50, 100, 13, 138}
	ZModemSZEnd = []byte{42, 42, 24, 66, 48, 56, 48, 48, 48, 48, 48, 48, 48, 48, 48, 50, 50, 100, 13, 138}
	// sz 结束后可能还会发送两个 OO，但是经过测试发现不一定每次都会发送 fmt.Sprintf("%+q", "OO")
	ZModemSZEndOO = []byte{79, 79}

	// rz fmt.Sprintf("%+q", "**\x18B0100000023be50\r\x8a\x11")
	ZModemRZStart = []byte{42, 42, 24, 66, 48, 49, 48, 48, 48, 48, 48, 48, 50, 51, 98, 101, 53, 48, 13, 138, 17}
	// rz -e fmt.Sprintf("%+q", "**\x18B0100000063f694\r\x8a\x11")
	ZModemRZEStart = []byte{42, 42, 24, 66, 48, 49, 48, 48, 48, 48, 48, 48, 54, 51, 102, 54, 57, 52, 13, 138, 17}
	// rz -S fmt.Sprintf("%+q", "**\x18B0100000223d832\r\x8a\x11")
	ZModemRZSStart = []byte{42, 42, 24, 66, 48, 49, 48, 48, 48, 48, 48, 50, 50, 51, 100, 56, 51, 50, 13, 138, 17}
	// rz -e -S fmt.Sprintf("%+q", "**\x18B010000026390f6\r\x8a\x11")
	ZModemRZESStart = []byte{42, 42, 24, 66, 48, 49, 48, 48, 48, 48, 48, 50, 54, 51, 57, 48, 102, 54, 13, 138, 17}
	// rz 结束 fmt.Sprintf("%+q", "**\x18B0800000000022d\r\x8a")
	ZModemRZEnd = []byte{42, 42, 24, 66, 48, 56, 48, 48, 48, 48, 48, 48, 48, 48, 48, 50, 50, 100, 13, 138}

	// **\x18B0
	ZModemRZCtrlStart = []byte{42, 42, 24, 66, 48}
	// \r\x8a\x11
	ZModemRZCtrlEnd1 = []byte{13, 138, 17}
	// \r\x8a
	ZModemRZCtrlEnd2 = []byte{13, 138}

	// zmodem 取消 \x18\x18\x18\x18\x18\x08\x08\x08\x08\x08
	ZModemCancel = []byte{24, 24, 24, 24, 24, 8, 8, 8, 8, 8}
)

func ByteContains(x, y []byte) (n []byte, contain bool)  {
	index := bytes.Index(x, y)
	if index == -1 {
		return
	}
	lastIndex := index + len(y)
	n = append(x[:index], x[lastIndex:]...)
	return n, true
}

// WebSSH 管理 Websocket 和 ssh 连接
type WebSSH struct {
	id string
	buffSize uint32
	term string
	sshConn net.Conn
	websocket *websocket.Conn
	connTimeout time.Duration
	logger *log.Logger
	DisableZModemSZ, DisableZModemRZ bool
	ZModemSZ, ZModemRZ, ZModemSZOO bool
}

// WebSSH 构造函数
func NewWebSSH() *WebSSH {
	return &WebSSH{
		buffSize: DefaultBuffSize,
		logger:   DefaultLogger,
		term: DefaultTerm,
		connTimeout: DefaultConnTimeout,
	}
}

func (ws *WebSSH) DisableSZ() {
	ws.DisableZModemSZ = true
}

func (ws *WebSSH) EnableSZ() {
	ws.DisableZModemSZ = false
}

func (ws *WebSSH) DisableRZ() {
	ws.DisableZModemRZ = true
}

func (ws *WebSSH) EnableRZ() {
	ws.DisableZModemRZ = false
}

func (ws *WebSSH) SetLogger(logger *log.Logger) {
	ws.logger = logger
}

// 设置 buff 大小
func (ws *WebSSH) SetBuffSize(buffSize uint32) {
	ws.buffSize = buffSize
}

// 设置日志输出
func (ws *WebSSH) SetLogOut(out io.Writer) {
	ws.logger.SetOutput(out)
}

// 设置终端 term 类型
func (ws *WebSSH) SetTerm(term string) {
	ws.term = term
}

// 设置连接 id
func (ws *WebSSH) SetId(id string) {
	ws.id = id
}

// 设置连接超时时间
func (ws *WebSSH) SetConnTimeOut(connTimeout time.Duration) {
	ws.connTimeout = connTimeout
}

// 添加 websocket 连接
func (ws *WebSSH) AddWebsocket(conn *websocket.Conn) {
	ws.logger.Printf("(%s) websocket connected", ws.id)
	ws.websocket = conn
	go func() {
		ws.logger.Printf("(%s) websocket exit %v", ws.id, ws.server())
	}()
}

// 添加 ssh 连接
func (ws *WebSSH) AddSSHConn(conn net.Conn) {
	ws.logger.Printf("(%s) ssh connected", ws.id)
	ws.sshConn = conn
}

// 处理 websocket 连接发送过来的数据
func (ws *WebSSH) server() error {
	defer func(){
		_ = ws.websocket.Close()
	}()

	// 默认加密方式 aes128-ctr aes192-ctr aes256-ctr aes128-gcm@openssh.com arcfour256 arcfour128
	// 连 linux 通常没有问题，但是很多交换机其实默认只提供 aes128-cbc 3des-cbc aes192-cbc aes256-cbc 这些。
	// 因此我们还是加全一点比较好。
	sshConfig := ssh.Config{
		Ciphers: []string{"aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com", "arcfour256", "arcfour128", "aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc"},
	}

	config := ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         ws.connTimeout,
		Config: sshConfig,
	}

	var session *ssh.Session
	var stdin io.WriteCloser
	var hasAddr bool
	var hasLogin bool
	var hasAuth bool
	var hasTerm bool

	for {
		var msg message
		//err := ws.websocket.ReadJSON(&msg)
		//if err != nil {
		//	return errors.Wrap(err, "websocket close or error message type")
		//}

		_, data, err := ws.websocket.ReadMessage()
		if err != nil {
			return errors.Wrap(err, "websocket close or read message err")
		}

		// 如果不是标准的信息格式，则是 xterm 输入或者 zmodem 数据流，则直接发送给 ssh 服务端
		err = json.Unmarshal(data, &msg)
		if err != nil {

			// fmt.Printf("ssh client input: %+q\n", string(data))

			_, err = stdin.Write(data)
			if err != nil {
				log.Println(err)
			}
		}
		switch msg.Type {
		case messageTypeIgnore:
			// 忽略的信息，比如使用 rz 时，记录里面无法看到上传的文件，
			// 客户端上传完成可以可以发个忽略信息过来让服务端知晓
			data, _ := url.QueryUnescape(string(msg.Data))
			fmt.Printf("Ignore message: %s", data)
		case messageTypeAddr:
			if hasAddr {
				continue
			}
			addr, _ := url.QueryUnescape(string(msg.Data))
			ws.logger.Printf("(%s) connect addr %s", ws.id, addr)
			conn, err := net.DialTimeout("tcp", addr, ws.connTimeout)
			if err != nil {
				_ = ws.websocket.WriteJSON(&message{Type: messageTypeStderr, Data: []byte("connect error\r\n")})
				return errors.Wrap(err, "connect addr " + addr + " error")
			}
			ws.AddSSHConn(conn)
			defer func() {
				_ = ws.sshConn.Close()
			}()
			hasAddr = true
		case messageTypeTerm:
			if hasTerm {
				continue
			}
			term, _ := url.QueryUnescape(string(msg.Data))
			ws.logger.Printf("(%s) set term %s", ws.id, term)
			ws.SetTerm(term)
			hasTerm = true
		case messageTypeLogin:
			if hasLogin {
				continue
			}
			config.User, _ = url.QueryUnescape(string(msg.Data))
			ws.logger.Printf("(%s) login with user %s", ws.id, config.User)
			hasLogin = true
		case messageTypePassword:
			if hasAuth {
				continue
			}

			if ws.sshConn == nil {
				ws.logger.Printf("must connect addr first")
				continue
			}

			if config.User == "" {
				ws.logger.Printf("must set user first")
				continue
			}

			password, _ := url.QueryUnescape(string(msg.Data))
			//ws.logger.Printf("(%s) auth with password %s", ws.id, password)
			ws.logger.Printf("(%s) auth with password ******", ws.id)
			config.Auth = append(config.Auth, ssh.Password(password))
			session, err = ws.newSSHXtermSession(ws.sshConn, &config, msg)
			if err != nil {
				_ = ws.websocket.WriteJSON(&message{Type: messageTypeStderr, Data: []byte("password login error\r\n")})
				return errors.Wrap(err, "password login error")
			}
			defer func() {
				_ = session.Close()
			}()

			stdin, err = session.StdinPipe()
			if err != nil {
				_ = ws.websocket.WriteJSON(&message{Type: messageTypeStderr, Data: []byte("get stdin channel error\r\n")})
				return errors.Wrap(err, "get stdin channel error")
			}
			defer func() {
				_ = stdin.Close()
			}()

			err = ws.transformOutput(session, ws.websocket)
			if err != nil {
				_ = ws.websocket.WriteJSON(&message{Type: messageTypeStderr, Data: []byte("get stdin & stderr channel error\r\n")})
				return errors.Wrap(err, "get stdin & stderr channel error")
			}

			err = session.Shell()
			if err != nil {
				_ = ws.websocket.WriteJSON(&message{Type: messageTypeStderr, Data: []byte("start a login shell error\r\n")})
				return errors.Wrap(err, "start a login shell error")
			}

			hasAuth = true

		case messageTypePublickey:
			if hasAuth {
				continue
			}

			if ws.sshConn == nil {
				ws.logger.Printf("must connect addr first")
				continue
			}

			if config.User == "" {
				ws.logger.Printf("must set user first")
				continue
			}

			//pemBytes, err := ioutil.ReadFile("/location/to/YOUR.pem")
			//if err != nil {
			//	return errors.Wrap(err, "publickey")
			//}

			// 传过来的 Data 是经过 url 编码的
			pemStrings, _ := url.QueryUnescape(string(msg.Data))
			//ws.logger.Printf("(%s) auth with privatekey %s", ws.id, pemStrings)
			ws.logger.Printf("(%s) auth with privatekey ******", ws.id)
			pemBytes := []byte(pemStrings)

			// 如果 key 有密码使用 ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(passphrase))
			signer, err := ssh.ParsePrivateKey(pemBytes)
			if err != nil {
				_ = ws.websocket.WriteJSON(&message{Type: messageTypeStderr, Data: []byte("parse publickey erro\r\n")})
				return errors.Wrap(err,"parse publickey error")
			}

			config.Auth = append(config.Auth, ssh.PublicKeys(signer))
			session, err = ws.newSSHXtermSession(ws.sshConn, &config, msg)
			if err != nil {
				_ = ws.websocket.WriteJSON(&message{Type: messageTypeStderr, Data: []byte("publickey login error\r\n")})
				return errors.Wrap(err, "publickey login error")
			}
			defer func() {
				_ = session.Close()
			}()

			stdin, err = session.StdinPipe()
			if err != nil {
				_ = ws.websocket.WriteJSON(&message{Type: messageTypeStderr, Data: []byte("get stdin channel error\r\n")})
				return errors.Wrap(err, "get stdin channel error")
			}
			defer func() {
				_ = stdin.Close()
			}()

			err = ws.transformOutput(session, ws.websocket)
			if err != nil {
				_ = ws.websocket.WriteJSON(&message{Type: messageTypeStderr, Data: []byte("get stdin & stderr channel error\r\n")})
				return errors.Wrap(err, "get stdin & stderr channel error")
			}
			err = session.Shell()
			if err != nil {
				_ = ws.websocket.WriteJSON(&message{Type: messageTypeStderr, Data: []byte("start a login shell error\r\n")})
				return errors.Wrap(err, "start a login shell error")
			}

			hasAuth = true

		// 为了兼容 zmodem， stdin 消息协议暂时无用，客户端数据都以二进制格式发送过来
		case messageTypeStdin:
			if stdin == nil {
				ws.logger.Printf("stdin wait login")
				continue
			}
			data, _ := url.QueryUnescape(string(msg.Data))
			_, err = stdin.Write([]byte(data))
			if err != nil {
				_ = ws.websocket.WriteJSON(&message{Type: messageTypeStderr, Data: []byte("write to stdin error\r\n")})
				return errors.Wrap(err, "write to stdin error")
			}

		case messageTypeResize:
			if session == nil {
				ws.logger.Printf("resize wait session")
				continue
			}
			err = session.WindowChange(msg.Rows, msg.Cols)
			if err != nil {
				_ = ws.websocket.WriteJSON(&message{Type: messageTypeStderr, Data: []byte("resize error\r\n")})
				return errors.Wrap(err, "resize error")
			}
		}
	}
}

// 创建 ssh 会话
func (ws *WebSSH) newSSHXtermSession(conn net.Conn, config *ssh.ClientConfig, msg message) (*ssh.Session, error) {
	// 也可以使用这种方法连接
	//client, err := ssh.Dial("tcp", "192.168.223.111:22", config)
	//if err != nil {
	//	return nil, errors.Wrap(err, "open client error")
	//}
	//session, err := client.NewSession()
	//if err != nil {
	//	return nil, errors.Wrap(err, "open session error")
	//}

	c, chans, reqs, err := ssh.NewClientConn(conn, conn.RemoteAddr().String(), config)
	if err != nil {
		return nil, errors.Wrap(err, "open client error")
	}
	session, err := ssh.NewClient(c, chans, reqs).NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "open session error")
	}
	modes := ssh.TerminalModes{
		ssh.ECHO: 1,
		ssh.TTY_OP_ISPEED: 8192,
		ssh.TTY_OP_OSPEED: 8192,
		ssh.IEXTEN: 0,
	}
	if msg.Cols <= 0 || msg.Cols > 500 {
		msg.Cols = 40
	}
	if msg.Rows <= 0 || msg.Rows > 1000 {
		msg.Rows = 80
	}
	err = session.RequestPty(ws.term, msg.Rows, msg.Cols, modes)
	if err != nil {
		return nil, errors.Wrap(err, "open pty error")
	}

	return session, nil
}

// 发送 ssh 会话的 stdout 和 stdin 数据到 websocket 连接
func (ws *WebSSH) transformOutput(session *ssh.Session, conn *websocket.Conn) error {
	stdout, err := session.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "get stdout channel error")
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "get stderr channel error")
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "get stdin channel error")
	}

	copyToMessage := func(t messageType, r io.Reader, w io.WriteCloser) {
		buff := make([]byte, ws.buffSize)
		for {
			n, err := r.Read(buff)
			if err != nil {
				return
			}

			//res := fmt.Sprintf("%+q", string(buff[:n]))
			//fmt.Println(buff[:n])
			//fmt.Println(t, res)

			if ws.ZModemSZOO {
				ws.ZModemSZOO = false
				if n < 2 {
					conn.WriteJSON(&message{Type: t, Data: buff[:n]})
				} else if n == 2 {
					if buff[0] == ZModemSZEndOO[0] && buff[1] == ZModemSZEndOO[1] {
						conn.WriteMessage(websocket.BinaryMessage, buff[:n])
					} else {
						conn.WriteJSON(&message{Type: t, Data: buff[:n]})
					}
				} else {
					if buff[0] == ZModemSZEndOO[0] && buff[1] == ZModemSZEndOO[1] {
						conn.WriteMessage(websocket.BinaryMessage, buff[:2])
						conn.WriteJSON(&message{Type: t, Data: buff[2:n]})
					} else {
						conn.WriteJSON(&message{Type: t, Data: buff[:n]})
					}
				}
			} else {
				if ws.ZModemSZ {
					if x, ok := ByteContains(buff[:n], ZModemSZEnd); ok {
						ws.ZModemSZ = false
						ws.ZModemSZOO = true
						conn.WriteMessage(websocket.BinaryMessage, ZModemSZEnd)
						if len(x) != 0 {
							conn.WriteJSON(&message{Type: messageTypeConsole, Data: x})
						}
					} else if _, ok := ByteContains(buff[:n], ZModemCancel); ok {
						ws.ZModemSZ = false
						conn.WriteMessage(websocket.BinaryMessage, buff[:n])
					} else {
						conn.WriteMessage(websocket.BinaryMessage, buff[:n])
					}
				} else if ws.ZModemRZ {
					if x, ok := ByteContains(buff[:n], ZModemRZEnd); ok {
						ws.ZModemRZ = false
						conn.WriteMessage(websocket.BinaryMessage, ZModemRZEnd)
						if len(x) != 0 {
							conn.WriteJSON(&message{Type: messageTypeConsole, Data: x})
						}
					} else if _, ok := ByteContains(buff[:n], ZModemCancel); ok {
						ws.ZModemRZ = false
						conn.WriteMessage(websocket.BinaryMessage, buff[:n])
					} else {
						// rz 上传过程中服务器端还是会给客户端发送一些信息，比如心跳
						//conn.WriteJSON(&message{Type: messageTypeConsole, Data: buff[:n]})
						//conn.WriteMessage(websocket.BinaryMessage, buff[:n])

						startIndex := bytes.Index(buff[:n], ZModemRZCtrlStart)
						if startIndex != -1 {
							endIndex := bytes.Index(buff[:n], ZModemRZCtrlEnd1)
							if endIndex != -1 {
								ctrl := append(ZModemRZCtrlStart, buff[startIndex + len(ZModemRZCtrlStart):endIndex]...)
								ctrl = append(ctrl, ZModemRZCtrlEnd1...)
								conn.WriteMessage(websocket.BinaryMessage, ctrl)
								info := append(buff[:startIndex], buff[endIndex + len(ZModemRZCtrlEnd1):n]...)
								if len(info) != 0 {
									conn.WriteJSON(&message{Type: messageTypeConsole, Data: info})
								}
							} else {
								endIndex = bytes.Index(buff[:n], ZModemRZCtrlEnd2)
								if endIndex != -1 {
									ctrl := append(ZModemRZCtrlStart, buff[startIndex + len(ZModemRZCtrlStart):endIndex]...)
									ctrl = append(ctrl, ZModemRZCtrlEnd2...)
									conn.WriteMessage(websocket.BinaryMessage, ctrl)
									info := append(buff[:startIndex], buff[endIndex + len(ZModemRZCtrlEnd2):n]...)
									if len(info) != 0 {
										conn.WriteJSON(&message{Type: messageTypeConsole, Data: info})
									}
								} else {
									conn.WriteJSON(&message{Type: messageTypeConsole, Data: buff[:n]})
								}
							}
						} else {
							conn.WriteJSON(&message{Type: messageTypeConsole, Data: buff[:n]})
						}
					}
				} else {
					if x, ok := ByteContains(buff[:n], ZModemSZStart); ok {
						if ws.DisableZModemSZ {
							conn.WriteJSON(&message{Type: messageTypeConsole, Data: []byte("sz download is disabled")})
							w.Write(ZModemCancel)
						} else {
							if y, ok := ByteContains(x, ZModemCancel); ok {
								// 下载不存在的文件以及文件夹(zmodem 不支持下载文件夹)时
								conn.WriteJSON(&message{Type: t, Data: y})
							} else {
								ws.ZModemSZ = true
								if len(x) != 0 {
									conn.WriteJSON(&message{Type: messageTypeConsole, Data: x})
								}
								conn.WriteMessage(websocket.BinaryMessage, ZModemSZStart)
							}
						}
					} else if x, ok := ByteContains(buff[:n], ZModemRZStart); ok {
						if ws.DisableZModemRZ {
							conn.WriteJSON(&message{Type: messageTypeConsole, Data: []byte("rz upload is disabled")})
							w.Write(ZModemCancel)
						} else {
							ws.ZModemRZ = true
							if len(x) != 0 {
								conn.WriteJSON(&message{Type: messageTypeConsole, Data: x})
							}
							conn.WriteMessage(websocket.BinaryMessage, ZModemRZStart)
						}
					} else if x, ok := ByteContains(buff[:n], ZModemRZEStart); ok {
						if ws.DisableZModemRZ {
							conn.WriteJSON(&message{Type: messageTypeConsole, Data: []byte("rz upload is disabled")})
							w.Write(ZModemCancel)
						} else {
							ws.ZModemRZ = true
							if len(x) != 0 {
								conn.WriteJSON(&message{Type: messageTypeConsole, Data: x})
							}
							conn.WriteMessage(websocket.BinaryMessage, ZModemRZEStart)
						}
					} else if x, ok := ByteContains(buff[:n], ZModemRZSStart); ok {
						if ws.DisableZModemRZ {
							conn.WriteJSON(&message{Type: messageTypeConsole, Data: []byte("rz upload is disabled")})
							w.Write(ZModemCancel)
						} else {
							ws.ZModemRZ = true
							if len(x) != 0 {
								conn.WriteJSON(&message{Type: messageTypeConsole, Data: x})
							}
							conn.WriteMessage(websocket.BinaryMessage, ZModemRZSStart)
						}
					} else if x, ok := ByteContains(buff[:n], ZModemRZESStart); ok {
						if ws.DisableZModemRZ {
							conn.WriteJSON(&message{Type: messageTypeConsole, Data: []byte("rz upload is disabled")})
							w.Write(ZModemCancel)
						} else {
							ws.ZModemRZ = true
							if len(x) != 0 {
								conn.WriteJSON(&message{Type: messageTypeConsole, Data: x})
							}
							conn.WriteMessage(websocket.BinaryMessage, ZModemRZESStart)
						}
					} else {
						conn.WriteJSON(&message{Type: t, Data: buff[:n]})
					}
				}
			}
		}
	}
	go copyToMessage(messageTypeStdout, stdout, stdin)
	go copyToMessage(messageTypeStderr, stderr, stdin)
	return nil
}
