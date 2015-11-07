package ggping

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	//Hold the maximum number a sequence can have. sequence will restart
	maxseq = 65536 //65536
)

type Ping struct {
	to       string            //hostname of the target host
	timeout  float64           //timeout to wait for a Pong at the pongchan
	interval float64           //The min interval between the iterations of this ping
	tos      int               //The Type od Service of this request
	ttl      int               //The Time to live packet
	requests int               //The number of ping requests we should send to the Target
	pktsz    uint              //The PacketSize to send
	usermap  map[string]string //User data for this ping
	err      error             //Set any error that appears before send the packet over the network
	when     time.Time         //The time when the last packet was sent. used as baseline to schedule the next packet to be send

	pongchan chan *Pong //The channel to wait for responses

	pongs []Pong //The received responses

	iphb    []byte        //Cache of marshalled ip header
	icmpmsg *icmp.Message //Stores the icmp message created at the first request
	toaddr  *net.IPAddr   //Stores the resolved ip address
	mu      sync.Mutex    //Used inside the Mutable functions
}

func NewPing(to string, timeout, interval float64, tos, ttl, requests int, pktsz uint, usermap map[string]string) (*Ping, error) {
	//Configure the Ping if it was not created by the NewPing function
	if to == "" {
		return nil, fmt.Errorf("to cannot be empty!")
	}
	if !(timeout > 0) {
		return nil, fmt.Errorf("Timeout sould be greater then 0")
	}
	if !(ttl > 0) {
		return nil, fmt.Errorf("Ttl should be a value greater than zero. use 64 is a good idea")
	}
	if !(requests > 0) {
		return nil, fmt.Errorf("Request should be greater than zero")
	}
	if !(pktsz > 0) {
		return nil, fmt.Errorf("Packet Size should be greater than zero")
	}

	p := Ping{
		to:       to,
		timeout:  timeout,
		interval: interval,
		tos:      tos,
		ttl:      ttl,
		requests: requests,
		pktsz:    pktsz,
		usermap:  usermap,
	}

	//Initializes the pongchan to receive EchoReplies from the pinger
	if p.pongchan == nil {
		p.pongchan = make(chan *Pong, 1)
	}

	return &p, nil
}

type Pong struct {
	EchoStart time.Time
	EchoEnd   time.Time
	Sequence  int
	err       error
}

//Transforms the request in a binary format to send over the network
//The binary form is made of the IP ehader + the Icmp Message
func (p *Ping) Marshall(id, seq int, clearCache bool) ([]byte, error) {

	//Prevents synchronization problems
	p.mu.Lock()
	defer p.mu.Unlock()

	debug.Printf("Value of pongchane %v\n", p.pongchan)

	//Clear internal cache
	if clearCache {
		p.toaddr = nil
		p.icmpmsg = nil
		p.iphb = nil
	}

	//Resolve the ipv4 ip address
	if p.toaddr == nil {
		if dst, err := net.ResolveIPAddr("ip4", p.to); err != nil {
			return nil, err
		} else {
			p.toaddr = dst
		}
	}

	//Build the Icmp Message
	if p.icmpmsg == nil {
		p.icmpmsg = &icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				Data: make([]byte, p.pktsz),
			},
		}
	}

	//Set the Identifier and the sequence
	p.icmpmsg.Body.(*icmp.Echo).ID = id & 0xffff
	p.icmpmsg.Body.(*icmp.Echo).Seq = seq

	//Marshall the icmp packet
	var icmpb []byte
	if b, err := p.icmpmsg.Marshal(nil); err != nil {
		return nil, err
	} else {
		icmpb = b
	}

	//Build the ip header
	if p.iphb == nil {
		iph := ipv4.Header{
			Version:  4,
			Len:      20,
			TOS:      p.tos,
			TotalLen: 20 + len(icmpb), // 20 bytes for IP, len(wb) for ICMP
			TTL:      p.ttl,
			Protocol: 1, // ICMP
			Dst:      p.toaddr.IP,
		}
		if iphb, err := iph.Marshal(); err != nil {
			return nil, err
		} else {
			p.iphb = iphb
		}
	}

	//Return the Marshalled packet, append ip header and icmp message
	return append(p.iphb, icmpb...), nil

}
func (p Pong) Rtt() (*time.Duration, error) {
	if p.err != nil {
		return nil, p.err
	}
	if (p.EchoEnd == time.Time{} || p.EchoStart == time.Time{}) {
		var buffer bytes.Buffer
		buffer.WriteString("The following time(s) are unset:")
		if (p.EchoEnd == time.Time{}) {
			buffer.WriteString(" [p.EchoEnd]")
		}
		if (p.EchoStart == time.Time{}) {
			buffer.WriteString(" [p.EchoStart]")
		}
		return nil, fmt.Errorf("%v", buffer.String())
	}
	t := p.EchoEnd.Sub(p.EchoStart)
	return &t, nil
}
