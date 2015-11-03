package ggping

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

//Send Icmp Packets over the network
//Receives request from the Ping Channel
//Register the new sequence in the pingarray that the receiver will send the timestamp
//Send Icmp Packet through the network
//Puts the Request back on the pong channel. The caller will verify the response and timeout
func pinger(ping chan Ping, pong chan Ping, pingInterval time.Duration) {

	//Maintains a sequence number
	var seq int

	//Initializes the slices that will receive the time channels
	pingarray := make([]chan time.Time, maxseq)

	//Creates the connection to send and receive packets
	c, err := net.ListenPacket("ip4:1", "0.0.0.0")
	if err != nil {
		log.Fatal("Could not open raw socket ip4:icmp: %v", err)
	}

	//defer c.Close()
	conn := ipv4.NewPacketConn(c)
	if err := conn.SetControlMessage(ipv4.FlagTTL|ipv4.FlagSrc|ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
		log.Fatal(err)
	}

	//Starts the icmp Listener in a goroutine
	//go receiver(conn, pingarray)
	go receiver(pingarray)

	//The engine loop
	for pi := range ping {

		//Increment the sequence number and assigns to pi.Seq
		seq = (seq + 1) % maxseq

		//Sets the sequence number into the binary packet
		//wb[6], wb[7] = uint8(seq>>8), uint8(seq&0xff)

		//Set the sequence number to the ping request
		pi.Seq = seq

		//Initializes the channel to receive the Pong
		pi.rttchan = make(chan time.Time, 1)

		//Registers the seq and the channel in the ping array
		//The listener will send a timestamp message to this channel
		//WHen the message arrives
		pingarray[seq] = pi.rttchan

		//Sets the sequence number into the binary packet
		//		wb[6], wb[7] = uint8(seq>>8), uint8(seq&0xff)
		//Creates the message to be sent based on Ping parameters
		wm := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Data: make([]byte, 56),
				Seq:  seq,
			},
		}
		//Encode the ICMP Packet into a binary representation
		wb, err := wm.Marshal(nil)
		if err != nil {
			log.Fatal(err)
		}

		//Saves the time before sending the packet
		pi.When = time.Now()

		//Send the packet over the Connection
		if _, err := conn.WriteTo(wb, nil, pi.toaddr); err != nil {
			pi.Pong = Pong{Err: fmt.Errorf("Could not send message through network")}
			pong <- pi
			return
		}

		//This  permits other goroutines to run and avoid saturate the socket with too
		//many simultaneous Echo Requests
		time.Sleep(pingInterval)

		//Returns the ping through the pong channel. The receiver will send a timestamp if a packet arrived for this sequence
		pong <- pi
	}
}
