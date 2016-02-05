package goping

import (
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

var (
	g_IDENTIFIER   uint8   = uint8(os.Getpid() & 0xff00) //Used to set the ID field of the ICMP Message
	g_SEQUENCE     counter = counter{max: 0x1000000}     //Used to set the next sequence field of the ICMP Message
	g_REQUESTID    counter                               //Used to set the next Ping ID
	g_PKT_SENT     counter
	g_PKT_RECEIVED counter
)

func getId() int {
	return int(uint16(g_IDENTIFIER)<<8 | uint16(g_SEQUENCE.uint64>>16))
}

//A Thread safe counter
type counter struct {
	uint64
	max uint64
	sync.Mutex
}

//The counter iterator
func (seq *counter) next() uint64 {
	seq.Lock()
	defer seq.Unlock()
	if seq.max > 0 {
		seq.uint64 = (seq.uint64 + 1) % seq.max
	} else {
		seq.uint64 = (seq.uint64 + 1)
	}
	return seq.uint64
}

//Represent an Echo Request to be sent over the network
type EchoRequest struct {
	WhenSent  time.Time      //The time this request was sent over the socket
	To        net.IPAddr     //The Ip Address of the request
	Size      uint8          //The number of bytes to send
	TOS       uint8          //The DSCP flag of the ip header
	TTL       uint8          //The TTL flag of the ip header
	Timeout   time.Duration  // The timeout
	chwhen    chan time.Time //THe channel to receive the time the ping was sent
	chreply   chan EchoReply //The Channel to receive the Echo Reply (Pong)
	chtimeout chan time.Time //The Channel to receive the Echo Reply (Pong)
}

//Represent an EchoReply received from the network
type EchoReply struct {
	WhenSent time.Time //When the Echo Request was sent
	WhenRecv time.Time //When the Echo Reply was received
	Type     uint8     //The type of the icmpReply Packet
	Code     uint8     //The Code for the Echo Reply
}

//The Response Object returned to the caller
type Pong struct {
	Ping         *Ping         //A reference to the Ping Request
	EchoRequests []EchoRequest //All the EchoRequests sent over the socket for the Ping Request
	EchoReplies  []EchoReply   //All the EchoReplies received from the socket for the Ping Request
}

//The Ping Request object
type Ping struct {
	To       string            //hostname of the target host
	Times    int               //The number of EchoRequests we should sent
	Timeout  time.Duration     //The duration we should wait for an EchoReply for each EchoRequest
	Interval time.Duration     //The duration we should wait between the sent EchoRequests
	TOS      int               //The Type of Service  to be sent in the EchoRequest IP Header
	TTL      int               //The Time to live to be sent in the  EchoRequest IP Header
	Size     uint              //The number of bytes to sent in each EchoRequest
	PeakTh   int               //The number of peaks to remove
	UserData map[string]string //User data

	Id uint64 //the id of this request. It is created  in the NewPing function and can be used to debug

	Toaddr     net.IPAddr //Stores the resolved ip address
	tosockaddr syscall.SockaddrInet4
}

func NewPing(to string) (*Ping, error) {
	//Configure the Ping if it was not created by the NewPing function
	if to == "" {
		return nil, fmt.Errorf("to cannot be empty!")
	}

	//Resolve the ipv4 ip address
	var dst *net.IPAddr
	var err error
	if dst, err = net.ResolveIPAddr("ip4", to); err != nil {
		return nil, err
	}
	ipb := dst.IP.To4()
	addrInet4 := syscall.SockaddrInet4{
		Port: 0,
		Addr: [4]byte{
			ipb[0],
			ipb[1],
			ipb[2],
			ipb[3],
		},
	}

	//Create a syscall.SockaddrInet4 for unix pingers

	return &Ping{
		Id:         g_REQUESTID.next(), //The track Id of this Ping Request
		To:         to,                 //The ping target received in this function
		Timeout:    3 * time.Second,    //Timeout default 3 seconds
		Interval:   1 * time.Second,    //Ping interval default 3 seconds
		TOS:        0,                  //Default TOS is 0
		TTL:        64,                 //Default TTL is 64
		Times:      1,                  //Default requests is 1
		Size:       54,                 //Default Packet Size is 54
		Toaddr:     *dst,
		tosockaddr: addrInet4,
	}, nil

}

//An Echo Request to be sent over the socket
func (p *Ping) makeEchoRequest() (EchoRequest, error) {

	return EchoRequest{
		To:        p.Toaddr,
		Timeout:   p.Timeout,
		Size:      uint8(p.Size),
		TOS:       uint8(p.TOS),
		TTL:       uint8(p.TTL),
		chreply:   make(chan EchoReply, 1), //The pinger will use this to send the EchoReply
		chwhen:    make(chan time.Time, 1), //The pinger will use this to send the Sent Time
		chtimeout: make(chan time.Time, 1), //The pinger will use this to send the TImeout Flag
	}, nil
}
