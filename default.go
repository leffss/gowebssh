package gowebssh

import (
	"io/ioutil"
	"log"
	"time"
)

var (
	DefaultTerm = TermXterm
	DefaultConnTimeout = 30 * time.Second
	DefaultLogger = log.New(ioutil.Discard, "[webssh] ", log.Ltime|log.Ldate)
	DefaultBuffSize = uint32(1024)
)