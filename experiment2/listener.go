package ggping

import (
	"log"
	"net"
	"time"

	"golang.org/x/net/icmp"
)

//Represents an ICMP ECHO Request
type Ping struct {
	To      string
	Timeout float64
	Err     error
	Pong    Pong

	ip   *net.IPAddr
	seq  int
	done chan bool
}

//Represents an ICMP ECHO Reply.
type Pong struct {
	When  time.Time
	Bytes []byte
	Size  int
	Peer  net.Addr
	Err   error
}

var (
	seq  int       //The sequence number to be used in the Ping requests
	pong chan Pong //Channel to receive Pong replies from the Raw Socket
	ping chan Ping //Channel to receive Ping requests from this api
)

func init() {

	//Initialize the ping channels
	ping = make(chan Ping)
	pong = make(chan Pong)

	//Starts an infinite loop to read from raw socket icmp
	go func() {
		//Opens the raw socket using the package golang/x/icmp from google
		c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		if err != nil {
			log.Fatal("Could not open raw socket ip4:icmp: %v", err)
		}

		for {
			//Reads an ICMP Message from the Socket.
			p := Pong{Bytes: make([]byte, 1500)}
			if p.Size, p.Peer, p.Err = c.ReadFrom(p.Bytes); p.Err != nil {
				log.Fatal("Could not read from socket: %v", err)
			}

			//Tags the time when the message arrived. This will be used to calc RTT
			p.When = time.Now()

			//Sends the Message to the pong channel
			go func(p Pong) {
				pong <- p
			}(p)
		}
	}()

	go func() {
		requests := make(map[int]chan<- string)
		for {
			select {
			case request := <-ping:

			case reply := <-ping:
			}
		}
	}()
}
