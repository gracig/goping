package lnxICMPv4

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/gracig/goping"
)

func init() {
	goping.RegPingerAdd("lnxICMPv4", new(pinger))
}

//Pinger is responsible for send and receive pings over the network
//type Pinger interface {
//	Ping(r Request, seq int) (future <-chan RawResponse,err error)
//}

type seqfut struct {
	seq int
	fut chan goping.RawResponse
}
type seqresp struct {
	seq  int
	resp goping.RawResponse
}

type pinger struct {
	sync.Once
	fd       int
	chseqfut chan seqfut
	chseqrep chan seqresp
	chdone   chan struct{}
}

func (p *pinger) Ping(r goping.Request, seq int) (future <-chan goping.RawResponse, err error) {

	//Get Future Channel
	fut := make(chan goping.RawResponse, 1)
	future = fut

	//Resolve HostName
	if addr, e := net.ResolveIPAddr("ip4", r.Host); e != nil {
		err = fmt.Errorf("Could not resolve address: %s", r.Host)
	} else {
		_ = addr
	}

	//Send seq and fut to a channel
	p.chseqfut <- seqfut{seq: seq, fut: fut}

	return
}

func (p *pinger) loop() {
	futures := make([]chan goping.RawResponse, 65536, 65536)
	for {
		select {
		case sf := <-p.chseqfut:
			futures[sf.seq] = sf.fut
		case sr := <-p.chseqrep:
			if futures[sr.seq] != nil {
				futures[sr.seq] <- sr.resp
				futures[sr.seq] = nil
			}
		case <-p.chdone:
			return
		}
	}
}

func (p *pinger) Init() {
	//Create a raw socket to read icmp packets
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	if err != nil {
		log.Fatalf("Could not open ICMP RAW Socket: [%v]\n", err)
	}

	//Set the option to receive the kernel timestamp from each received message
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_TIMESTAMP, 1); err != nil {
		log.Fatalf("Could not set sock opt syscall [%v]\n", err)
	}

	//Increase the socket buffer
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, 1024*1024); err != nil {
		log.Fatalf("Could not set sock opt syscall Increase buffer: [%v]\n", err)
	}

	//Listen on all interfaces
	var addr syscall.Sockaddr = &syscall.SockaddrInet4{
		Port: 0,
		Addr: [4]byte{0, 0, 0, 0},
	}
	if err := syscall.Bind(fd, addr); err != nil {
		log.Fatalf("Could not bind all interfaces to socket: [%v]", err)
	}

	p.fd = fd
	go p.loop()
	go p.startListener(fd)

}

func (p *pinger) Close() {
	if err := syscall.Close(p.fd); err != nil {
		log.Printf("Error while closing socket: [%v]", err)
	}
}

func (p *pinger) startListener(fd int) {

	//Buffer to receive the ping packet
	buf := make([]byte, 1024)

	//Buffer to receive the control message
	oob := make([]byte, 64)

	//Variables used in the for loop
	var pid, seq int
	var tv syscall.Timeval
	var bbuf bytes.Buffer

	for {
		//Receives a message from the socket sent by the kernel
		_, oobn, _, peer, err := syscall.Recvmsg(fd, buf, oob, 0)
		if err != nil {
			log.Printf("Error reading icmp packet from the socket: %v\n", err)
			return //Continue on error
		}

		//Finds the pid and seq value.
		switch buf[20] {
		case 8:
			//Received an echo request. Probably from localhost. ignoring
			continue
		case 0:
			//Received an Echo Reply
			pid = int(uint16(buf[24])<<8 | uint16(buf[25]))
			seq = int(uint16(buf[26])<<8 | uint16(buf[27]))
		default:
			//Received an error message
			pid = int(uint16(buf[52])<<8 | uint16(buf[53]))
			seq = int(uint16(buf[54])<<8 | uint16(buf[55]))
			//emsg = msg[28:] //20+28+4
		}

		//Ignores processing if id is different from mypid
		if pid != g_IDENTIFIER {
			continue
		}

		//Find the pinger object related to the Sequence value in the g_PINGER_RING
		findPinger(&ps, seq)
		if ps.r.Value == nil {
			info.Println("Could not find pinger for sequence:", seq)
			continue
		}
		p := ps.r.Value.(*pinger)
		if p == nil {
			info.Println("Could not receive pinger from g_REPLY_ARRAY")
			continue
		}
		p.reply.WhenRecv = time.Now()
		p.reply.Peer = net.IPv4(
			peer.(*syscall.SockaddrInet4).Addr[0],
			peer.(*syscall.SockaddrInet4).Addr[1],
			peer.(*syscall.SockaddrInet4).Addr[2],
			peer.(*syscall.SockaddrInet4).Addr[3],
		)
		p.reply.ICMPType = buf[20]
		p.reply.ICMPCode = buf[21]

		//Parses the Control Message to find the SO_TIMESTAMP value
		cmsgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
		if err != nil {
			severe.Println(os.NewSyscallError("parse socket control message", err))
		}

		//Iterate over the control messages
		for _, m := range cmsgs {
			//Continue if control message is not syscall.SOL_SOCKET
			if m.Header.Level != syscall.SOL_SOCKET {
				continue
			}
			//Control Message is SOL_SOCKET, Verifyng if syscall is SO_TIMESTAMP
			switch m.Header.Type {
			case syscall.SO_TIMESTAMP:
				bbuf.Write(m.Data)
				binary.Read(&bbuf, binary.LittleEndian, &tv)
				bbuf.Reset()
				p.reply.WhenRecv = time.Unix(tv.Unix())
			}
		}

		//Send the pong object  back to the caller
		go func(p *pinger) {
			p.chPong <- p
			debug.Println("Sent pong reference to the ping.pongchan reference ")
		}(p)
	}

}
