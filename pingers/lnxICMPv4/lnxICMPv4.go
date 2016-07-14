package lnxICMPv4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"math"
	"net"
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

type ConnManager interface {
	Open() (int, error)
	Close(fd int) error
	SendTo(fd int, p []byte, to syscall.Sockaddr) error
	RecvMsg(fd int, buf []byte) (peer net.IP, when time.Time, err error)
}

type pinger struct {
	conn ConnManager
}

func (p *pinger) Start(pid int) (chan<- goping.SeqRequest, <-chan goping.RawResponse, <-chan struct{}, error) {
	in, out, donein, done := make(chan goping.SeqRequest), make(chan goping.RawResponse), make(chan struct{}), make(chan struct{})
	if fd, err := p.conn.Open(); err != nil {
		return nil, nil, nil, err
	} else {
		go p.ping(pid, fd, in, out, donein)
		go p.pong(pid, fd, out, donein, done)
	}

	return in, out, done, nil
}

func (p *pinger) ping(gpid int, fd int, in <-chan goping.SeqRequest, out chan<- goping.RawResponse, done chan<- struct{}) {

	for r := range in {

		//Resolve HostName
		if addr, e := net.ResolveIPAddr("ip4", r.Req.Host); e != nil {
			out <- goping.RawResponse{Seq: r.Seq, Err: errors.New("Could not resolve address"), RTT: math.NaN()}

		} else {
			_ = addr
		}

	}
	done <- struct{}{}

}
func (p *pinger) pong(gpid int, fd int, out chan<- goping.RawResponse, donein <-chan struct{}, done chan<- struct{}) {

	//Buffer to receive the ping packet
	buf := make([]byte, 1024)

	//Variables used in the for loop
	var pid, seq int

	for {
		//Receives a message from the socket sent by the kernel

		peer, when, err := p.conn.RecvMsg(fd, buf)
		if err != nil {
			log.Printf("Error reading icmp packet from the socket: %v\n", err)
			continue //Continue on error
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
		if pid != gpid {
			continue
		}

		//Send the raw response back to channel
		out <- goping.RawResponse{Seq: seq, ICMPMessage: buf[:40], Peer: peer, RTT: 0}

	}

}

/***Conn Manager Implementation ***/
type connmanager struct{}

func (c *connmanager) Open() (int, error) {
	//Create a raw socket to read icmp packets
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	if err != nil {
		return 0, err
	}

	//Set the option to receive the kernel timestamp from each received message
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_TIMESTAMP, 1); err != nil {
		return 0, err
	}

	//Increase the socket buffer
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, 1024*1024); err != nil {
		return 0, err
	}

	//Listen on all interfaces
	var addr syscall.Sockaddr = &syscall.SockaddrInet4{
		Port: 0,
		Addr: [4]byte{0, 0, 0, 0},
	}
	if err := syscall.Bind(fd, addr); err != nil {
		return 0, err
	}
	return fd, nil

}
func (c *connmanager) Close(fd int) error {
	return syscall.Close(fd)
}

func (c *connmanager) SendTo(fd int, p []byte, to syscall.Sockaddr) error {
	return syscall.Sendto(fd, p, 0, to)
}
func (c *connmanager) RecvFrom(fd int, buf []byte) (peer net.IP, when time.Time, rerr error) {

	//Buffer to receive the control message
	oob := make([]byte, 64)

	//Receive Message from socket
	n, oobn, recvflags, from, err := syscall.Recvmsg(fd, buf, oob, 0)
	if err != nil {
		rerr = err
		return
	}

	//Parses the Control Message to find the SO_TIMESTAMP value
	if cmsgs, err := syscall.ParseSocketControlMessage(oob[:oobn]); err != nil {
		rerr = err
		return
	} else {

		var bbuf bytes.Buffer
		var tv syscall.Timeval
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
				when = time.Unix(tv.Unix())
			}
		}
	}

	//Get peer address
	peer = net.IPv4(
		from.(*syscall.SockaddrInet4).Addr[0],
		from.(*syscall.SockaddrInet4).Addr[1],
		from.(*syscall.SockaddrInet4).Addr[2],
		from.(*syscall.SockaddrInet4).Addr[3],
	)

	return

}
