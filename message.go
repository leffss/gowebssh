package gowebssh

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
	Cols int         `json:"cols,omitempty"`
	Rows int         `json:"rows,omitempty"`
}
