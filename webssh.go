package gowebssh

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// NewWebSSH 新建对象
func NewWebSSH() *WebSSH {
	return &WebSSH{
		buffSize: 1024,
		expired:  5 * time.Minute,
		logger:   log.New(ioutil.Discard, "[webssh] ", log.Ltime|log.Ldate),
		term: "xterm",
	}
}

// WebSSH Websocket和ssh
type WebSSH struct {
	logger   *log.Logger
	store    sync.Map
	expired  time.Duration
	buffSize uint32
	term string
}

type storeValue struct {
	id        string
	websocket *websocket.Conn
	conn      net.Conn
	createdAt time.Time
}

// SetLogger set logger
func (ws *WebSSH) SetLogger(logger *log.Logger) *WebSSH {
	ws.logger = logger
	return ws
}

// SetBuffSize set buff size
func (ws *WebSSH) SetBuffSize(buffSize uint32) *WebSSH {
	ws.buffSize = buffSize
	return ws
}

// SetLogOut set logger output
func (ws *WebSSH) SetLogOut(out io.Writer) *WebSSH {
	ws.logger.SetOutput(out)
	return ws
}

// SetExpired set logger
func (ws *WebSSH) SetExpired(expired time.Duration) *WebSSH {
	ws.expired = expired
	return ws
}

// SetExpired set logger
func (ws *WebSSH) SetTerm(term string) *WebSSH {
	ws.term = term
	return ws
}

// AddWebsocket add websocket connect
func (ws *WebSSH) AddWebsocket(id string, conn *websocket.Conn) {
	ws.logger.Println("add websocket", id)

	ws.checkExpired()
	v, loaded := ws.store.LoadOrStore(id, &storeValue{websocket: conn, id: id, createdAt: time.Now()})
	if !loaded {
		return
	}
	value := v.(*storeValue)
	value.websocket = conn
	go func() {
		ws.logger.Printf("%s server exit %v", id, ws.server(value))
	}()
}

// AddSSHConn add ssh netword connect
func (ws *WebSSH) AddSSHConn(id string, conn net.Conn) {
	ws.logger.Println("add ssh conn", id)

	ws.checkExpired()
	v, loaded := ws.store.LoadOrStore(id, &storeValue{conn: conn, id: id, createdAt: time.Now()})
	if !loaded {
		return
	}
	value := v.(*storeValue)
	value.conn = conn
	go func() {
		ws.logger.Printf("(%s) server exit %v", id, ws.server(value))
	}()
}

func (ws *WebSSH) checkExpired() {
	now := time.Now()
	ws.store.Range(func(key, v interface{}) bool {
		value := v.(*storeValue)
		if now.Sub(value.createdAt) > time.Minute {
			ws.store.Delete(key)
			if value.websocket != nil {
				value.websocket.Close()
			}
			if value.conn != nil {
				value.conn.Close()
			}
		}
		return true
	})
}

// server connect ssh conn to websocket
func (ws *WebSSH) server(value *storeValue) error {
	ws.store.Delete(value.id)
	ws.logger.Println("server", value)
	defer value.websocket.Close()
	defer value.conn.Close()

	config := ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	var session *ssh.Session
	var stdin io.WriteCloser
	for {
		var msg message
		err := value.websocket.ReadJSON(&msg)
		if err != nil {
			return errors.Wrap(err, "login")
		}
		ws.logger.Println("new message", msg.Type)
		switch msg.Type {
		case messageTypeLogin:
			ws.logger.Printf("login %s", msg.Data)
			config.User, _ = url.QueryUnescape(string(msg.Data))
		case messageTypePassword:
			password, _ := url.QueryUnescape(string(msg.Data))
			config.Auth = append(config.Auth, ssh.Password(password))
			session, err = ws.newSSHXtermSession(value.conn, &config, msg)
			if err != nil {
				return errors.Wrap(err, "password")
			}
			defer session.Close()
			stdin, err = session.StdinPipe()
			if err != nil {
				return errors.Wrap(err, "stdin")
			}
			defer stdin.Close()
			err = ws.transformOutput(session, value.websocket)
			if err != nil {
				return errors.Wrap(err, "stdout & stderr")
			}
			err = session.Shell()
			if err != nil {
				return errors.Wrap(err, "shell")
			}

		case messageTypePublickey:
			//pemBytes, err := ioutil.ReadFile("/location/to/YOUR.pem")
			//if err != nil {
			//	return errors.Wrap(err, "publickey")
			//}

			// 传过来的 Data 是进过 url 编码的
			pemStrings, _ := url.QueryUnescape(string(msg.Data))
			pemBytes := []byte(pemStrings)

			signer, err := ssh.ParsePrivateKey(pemBytes)
			if err != nil {
				return errors.Wrap(err,"parse publickey failed")
			}

			config.Auth = append(config.Auth, ssh.PublicKeys(signer))
			session, err = ws.newSSHXtermSession(value.conn, &config, msg)
			if err != nil {
				return errors.Wrap(err, "publickey")
			}
			defer session.Close()
			stdin, err = session.StdinPipe()
			if err != nil {
				return errors.Wrap(err, "stdin")
			}
			defer stdin.Close()
			err = ws.transformOutput(session, value.websocket)
			if err != nil {
				return errors.Wrap(err, "stdout & stderr")
			}
			err = session.Shell()
			if err != nil {
				return errors.Wrap(err, "shell")
			}

		case messageTypeStdin:
			if stdin == nil {
				ws.logger.Println("stdin wait login")
				continue
			}
			_, err = stdin.Write(msg.Data)
			if err != nil {
				return errors.Wrap(err, "write")
			}
		case messageTypeResize:
			if session == nil {
				ws.logger.Println("resize wait session")
				continue
			}
			err = session.WindowChange(msg.Rows, msg.Cols)
			if err != nil {
				return errors.Wrap(err, "resize")
			}
		}
	}
}

// newSSHXtermSession start ssh xterm session
func (ws *WebSSH) newSSHXtermSession(conn net.Conn, config *ssh.ClientConfig, msg message) (*ssh.Session, error) {
	var err error
	c, chans, reqs, err := ssh.NewClientConn(conn, conn.RemoteAddr().String(), config)
	if err != nil {
		return nil, errors.Wrap(err, "client")
	}
	session, err := ssh.NewClient(c, chans, reqs).NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "session")
	}
	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: ws.buffSize, ssh.TTY_OP_OSPEED: ws.buffSize}
	if msg.Cols == 0 {
		msg.Cols = 40
	}
	if msg.Rows == 0 {
		msg.Rows = 80
	}
	session.RequestPty(ws.term, msg.Rows, msg.Cols, modes)
	return session, nil
}

// transformOutput transform shell stdout to websocket message
func (ws *WebSSH) transformOutput(session *ssh.Session, conn *websocket.Conn) error {
	ws.logger.Println("transfer")
	stdout, err := session.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "stdout")
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		errors.Wrap(err, "stderr")
	}
	copyToMessage := func(t messageType, r io.Reader) {
		ws.logger.Println("copy to", t)
		buff := make([]byte, ws.buffSize)
		for {
			n, err := r.Read(buff)
			if err != nil {
				ws.logger.Printf("%s read fail", t)
				return
			}
			err = conn.WriteJSON(&message{Type: t, Data: buff[:n]})
			if err != nil {
				ws.logger.Printf("%s write fail", t)
				return
			}
		}
	}

	go copyToMessage(messageTypeStdout, stdout)
	go copyToMessage(messageTypeStderr, stderr)

	return nil
}