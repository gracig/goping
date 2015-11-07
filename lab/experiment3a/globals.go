package ggping

import (
	"net"
	"time"
)

const (
	//Hold the maximum number a sequence can have. sequence will restart
	maxseq = 65536 //65536
)

//Creates a map to match requests with a channel to send response

type Ping struct {
	To      string
	Timeout uint
	EchoMap map[string]string
	When    time.Time
	Seq     int
	Pong    Pong

	rttchan chan time.Time
	toaddr  *net.IPAddr
}

type Pong struct {
	Rtt float64
	Err error
}
