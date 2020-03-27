package gowebssh

type messageType string

const (
	messageTypeAddr		 = "addr"
	messageTypeTerm      = "term"
	messageTypeLogin     = "login"
	messageTypePassword  = "password"
	messageTypePublickey = "publickey"
	messageTypeStdin     = "stdin"
	messageTypeStdout    = "stdout"
	messageTypeStderr    = "stderr"
	messageTypeResize    = "resize"
	messageTypeIgnore    = "ignore"
)

type message struct {
	Type messageType `json:"type"`
	Data []byte      `json:"data"`
	Cols int         `json:"cols,omitempty"`
	Rows int         `json:"rows,omitempty"`
}