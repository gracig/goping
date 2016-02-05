package goping

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var (
	g_IDENTIFIER int     = os.Getpid() & 0xffff
	g_SEQUENCE   counter = counter{max: 65536}
	g_REQUESTID  counter = counter{max: 100000000}
)

type counter struct {
	int64
	max int64
	sync.Mutex
}

func (seq *counter) next() int64 {
	seq.Lock()
	defer seq.Unlock()
	seq.int64 = (seq.int64 + 1) % seq.max
	return seq.int64
}

type Summary struct {
	Pongs      []*Pong //The received responses
	PingTask   *PingTask
	Min        float64
	Max        float64
	Avg        float64
	Sent       float64
	Success    float64
	Failed     float64
	PacketLoss float64
	RttSum     float64
}

type EchoRequest struct {
	pingTask *PingTask   //Reference to the PingTask the generated this Echo Request
	to       *net.IPAddr //The Ip Address of the request
	when     time.Time   //The time when this request was created
	sequence int         //The icmp Sequence field
	index    int         //The index to refer to the pongchan[] when[] and pongs[] arrays on the Pong Task
	bipv4    []byte      //The marshalled ipv4 request
	bicmp    []byte      //The marshalled icmp echo request
	pongchan chan *Pong  //The Channel to receive the Echo Reply (Pong)
	err      error       //Register errors before send the packet over the network
	next     bool
}

type PingTask struct {
	to       string            //hostname of the target host
	timeout  time.Duration     //timeout to wait for a Pong at the pongchan
	interval time.Duration     //The min interval between the iterations of this ping
	tos      int               //The Type od Service of this request
	ttl      int               //The Time to live packet
	requests int               //The number of ping requests we should send to the Target
	pktsz    uint              //The PacketSize to send
	usermap  map[string]string //User data for this ping
	err      error             //Set any error that appears before send the packet over the network
	received int               //The number of sucessful replies

	id      int64   //the id of this request
	counter counter //the actual index of the Ping Task. This go from 1 to number of requests

	pongchan []chan *Pong //The channels to wait for responses
	when     []time.Time  //The time when the last packet was sent. used as baseline to schedule the next request
	//	pongstatus []byte       //The  pong status
	//	pongrtt    []float64    //The  pong response

	iphb    []byte        //Cache of marshalled ip header
	icmpmsg *icmp.Message //Stores the icmp message created at the first request
	toaddr  *net.IPAddr   //Stores the resolved ip address
	mu      sync.Mutex    //Used inside the Mutable functions
}

func NewPingTask(to string, timeout, interval time.Duration, tos, ttl, requests int, pktsz uint, usermap map[string]string) (*PingTask, error) {
	//Configure the Ping if it was not created by the NewPing function
	if to == "" {
		return nil, fmt.Errorf("to cannot be empty!")
	}
	if !(timeout > 0) {
		return nil, fmt.Errorf("Timeout should be greater then 0")
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

	p := PingTask{
		id:       g_REQUESTID.next(),
		to:       to,
		timeout:  timeout,
		interval: interval,
		tos:      tos,
		ttl:      ttl,
		requests: requests,
		pktsz:    pktsz,
		usermap:  usermap,
		counter:  counter{max: int64(requests + 1)},
	}

	//Init the channel to receive pongs for each request
	p.pongchan = make([]chan *Pong, requests)
	for i := 0; i < requests; i++ {
		p.pongchan[i] = make(chan *Pong, 1)
	}
	//Init the slice to store the time the packet was sent
	p.when = make([]time.Time, requests)

	return &p, nil
}

type Pong struct {
	EchoStart time.Time
	EchoEnd   time.Time
	Err       error

	iph  []byte
	imh  []byte
	peer []byte

	pi  *PingTask
	idx int
}

func (this Pong) String() string {
	if this.Err != nil {
		return fmt.Sprintf("ERROR from %v (%v): req=%d seq=%d time=%.3f ms %v %v",
			this.pi.to,
			this.pi.toaddr,
			this.pi.id,
			this.idx+1,
			float64(this.EchoEnd.Sub(this.EchoStart))/float64(time.Millisecond),
			this.Err,
			this.pi.pongchan[this.idx],
		)
	}

	return fmt.Sprintf("%v bytes from %v (%v): req=%d seq=%d ttl=%d time=%.3f ms",
		len(this.imh),
		this.pi.to,
		this.pi.toaddr,
		this.pi.id,
		this.idx+1,
		int(uint8(this.iph[8])),
		float64(this.EchoEnd.Sub(this.EchoStart))/float64(time.Millisecond))
}

func (this *PingTask) appendPong(p *Pong, index int) {
	//this.pongs[index] = p
}

//Transforms the request in a binary format to send over the network
//The binary form is made of the IP ehader + the Icmp Message
func (p *PingTask) makeEchoRequest(clearCache bool) (*EchoRequest, error) {

	//Prevents synchronization problems
	p.mu.Lock()
	defer p.mu.Unlock()

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
	sequence := int(g_SEQUENCE.next())
	p.icmpmsg.Body.(*icmp.Echo).ID = g_IDENTIFIER
	p.icmpmsg.Body.(*icmp.Echo).Seq = sequence

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
	//Increment the p.counter
	index := p.counter.next() - 1
	return &EchoRequest{
		pingTask: p,
		when:     time.Now(),
		sequence: sequence,
		to:       p.toaddr,
		index:    int(index),
		bipv4:    p.iphb,
		bicmp:    icmpb,
		pongchan: p.pongchan[index],
	}, nil

}
func (p Pong) Rtt() (*time.Duration, error) {
	if p.Err != nil {
		return nil, p.Err
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
