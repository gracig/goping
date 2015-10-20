package ggping

import (
	"log"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type rawIcmp struct {
	when    time.Time
	size    int
	peer    net.Addr
	bytes   []byte
	cm      *ipv4.ControlMessage
	message *icmp.Echo //The message after being parsed
	err     error
}

func runListener(handleRawIcmp func(ri *rawIcmp)) {
	//Starts an infinite loop to read from raw socket icmp
	go func() {

		c, err := net.ListenPacket("ip4:1", "0.0.0.0")
		if err != nil {
			log.Fatal("Could not open raw socket ip4:icmp: %v", err)
		}
		defer c.Close()
		p := ipv4.NewPacketConn(c)
		if err := p.SetControlMessage(ipv4.FlagTTL|ipv4.FlagSrc|ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
			log.Fatal(err)
		}

		for {
			//Reads an ICMP Message from the Socket.
			ri := rawIcmp{bytes: make([]byte, 1500)}
			if ri.size, ri.cm, ri.peer, ri.err = p.ReadFrom(ri.bytes); ri.err != nil {
				log.Fatal("Could not read from socket: %v", ri.err)
			}

			//Tags the time when the message arrived. This will be used to calc RTT
			ri.when = time.Now()

			//Sends the Message to the checho channel
			go func(r rawIcmp) {
				handleRawIcmp(&r)
			}(ri)
		}
	}()
}
