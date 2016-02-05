package goping

import (
	"fmt"
	"log"
	"os"
	"syscall"
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

	//Create a raw socket to read icmp packets
	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	//Tells the socket we will build the ip header
	if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
		log.Fatal("Could not set sock opt syscall HDRINCL")
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

		//Uses the ipv4 and icmp packets to help build the packets
		//Builds the icmp message
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

		//Builds the ip Header
		iph := ipv4.Header{
			Version:  4,
			Len:      20,
			TOS:      0,
			TotalLen: 20 + len(wb), // 20 bytes for IP, len(wb) for ICMP
			TTL:      64,
			Protocol: 1, // ICMP
			Dst:      pi.toaddr.IP,
		}
		ipb, err := iph.Marshal()
		if err != nil {
			log.Fatal(err)
		}

		//Builds the packet append ip header and icmp message

		pkt := append(ipb, wb...)

		//Builds the target SockaddrInet4
		ipb = pi.toaddr.IP.To4()
		to := syscall.SockaddrInet4{
			Port: 0,
			Addr: [4]byte{
				ipb[0],
				ipb[1],
				ipb[2],
				ipb[3],
			},
		}

		//Setting the time before send the packet
		pi.When = time.Now()

		//Sending the packet through the network
		err = syscall.Sendto(fd, pkt, 0, &to)
		if err != nil {
			fmt.Println("Error Sending Message", err)
			pi.Pong = Pong{Err: err}
		}

		//This  permits other goroutines to run and avoid saturate the socket with too
		//many simultaneous Echo Requests
		time.Sleep(pingInterval)

		//Returns the ping through the pong channel. The receiver will send a timestamp if a packet arrived for this sequence
		pong <- pi
	}
}
func FD_SET(p *syscall.FdSet, i int) {
	p.Bits[i/64] |= 1 << uint(i) % 64
}

func FD_ISSET(p *syscall.FdSet, i int) bool {
	return (p.Bits[i/64] & (1 << uint(i) % 64)) != 0
}

func FD_ZERO(p *syscall.FdSet) {
	for i := range p.Bits {
		p.Bits[i] = 0
	}
}
