package ggping

import (
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var (
	g_IDENTIFIER int     = os.Getpid() & 0xffff //Used to set the ID field of the ICMP Message
	g_SEQUENCE   counter = counter{max: 65536}  //Used to set the next sequence field of the ICMP Message
	g_REQUESTID  counter                        //Used to set the next Ping ID
)

//A Thread safe counter
type counter struct {
	int64
	max int64
	sync.Mutex
}

//The counter iterator
func (seq *counter) next() int64 {
	seq.Lock()
	defer seq.Unlock()
	if seq.max > 0 {
		seq.int64 = (seq.int64 + 1) % seq.max
	} else {
		seq.int64 = (seq.int64 + 1)
	}
	return seq.int64
}

//Represent an Echo Request to be sent over the network
type EchoRequest struct {
	Id         int64      //The id of the ping request
	To         net.IPAddr //The Ip Address of the request
	When       time.Time  //The time when this request was sent over the socket
	Err        error      //Register errors before send the packet over the network
	tosockaddr *syscall.SockaddrInet4

	bipv4   []byte         //The marshalled ipv4 request
	bicmp   []byte         //The marshalled icmp echo request
	chreply chan EchoReply //The Channel to receive the Echo Reply (Pong)
	chwhen  chan time.Time //THe channel to receive the time the ping was generated
}

func (e EchoRequest) GetSequence() int {
	if len(e.bicmp) > 8 {
		return int(uint16(e.bicmp[6])<<8 | uint16(e.bicmp[7]))
	}
	return 0
}

//Represent an EchoReply received from the network
type EchoReply struct {
	From net.IP    //The ip address that answered the Echo Request
	When time.Time //When the Echo Reply was received
	Err  error     //The error, if any

	iph []byte //THe bytes received in the IP Header
	imh []byte //The bytes received in the ICMP Message
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

	Id int64 //the id of this request. It is created  in the NewPing function and can be used to debug

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

	//Build the Icmp Message
	icmpmsg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			Data: make([]byte, p.Size),
			ID:   g_IDENTIFIER,
			Seq:  int(g_SEQUENCE.next()),
		},
	}

	//Marshall the icmp packet
	var err error
	var icmpb []byte
	if icmpb, err = icmpmsg.Marshal(nil); err != nil {
		return EchoRequest{}, err
	}

	//Build the ip header
	iph := ipv4.Header{
		Version:  4,
		Len:      20,
		TOS:      p.TOS,
		TotalLen: 20 + len(icmpb), // 20 bytes for IP, len(wb) for ICMP
		TTL:      p.TTL,
		Protocol: 1, // ICMP
		Dst:      p.Toaddr.IP,
	}

	var iphb []byte
	if iphb, err = iph.Marshal(); err != nil {
		return EchoRequest{}, err
	}

	//Return the Marshalled packet, append ip header and icmp message
	//Increment the p.counter
	return EchoRequest{
		Id:         p.Id,
		When:       time.Now(), //May be modified by the Pinger
		To:         p.Toaddr,
		tosockaddr: &p.tosockaddr,

		bipv4:   iphb,                    //The ping will use this as the IPv4 Header
		bicmp:   icmpb,                   //The pinger will use this as the Icmp Message to send
		chreply: make(chan EchoReply, 1), //The pinger will use this to send the EchoReply
		chwhen:  make(chan time.Time, 1), //The pinger will use this to send the EchoReply
	}, nil
}
