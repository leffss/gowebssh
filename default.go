package gowebssh

import (
	"io/ioutil"
	"log"
	"time"
)

var (
	DefaultTerm = TermXterm
	DefaultConnTimeout = 15 * time.Second
	DefaultLogger = log.New(ioutil.Discard, "[webssh] ", log.Ltime|log.Ldate)
	DefaultBuffSize = uint32(8192)
)