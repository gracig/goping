package linuxICMPv4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"syscall"
	"time"
	"unsafe"

	"github.com/gracig/goping"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func New() goping.Pinger {
	return &pinger{syscall: new(syscallWrapper)}
}

//Pinger is the type the implements goping.Pinger interface
type pinger struct {
	syscall syscallWrapperInterface //The syscall Wrapper object
}

//Start is the implementation of the method goping.Pinger.Start
func (p pinger) Start(pid int) (ping chan<- goping.SeqRequest, pong <-chan goping.RawResponse, done <-chan struct{}, err error) {

	//Initialize the channels used in the select stage
	input, output, doneInput, doneOutput := make(chan goping.SeqRequest), make(chan goping.RawResponse), make(chan struct{}), make(chan struct{})

	//Opens the connection
	if fd, err := p.OpenConn(); err != nil {
		//Returns error that connection could not be opened
		return nil, nil, nil, fmt.Errorf("Connection could not be opened: %v", err)
	} else {
		//Start Sending ICMPRequests to the File Descriptor
		go p.ping(pid, fd, input, output, doneInput)
		//Start receiving ICMPReplies from the File Descriptor
		go p.pong(pid, fd, output, doneInput, doneOutput)
	}

	ping, pong, done = input, output, doneOutput
	return
}

//Opens a raw socket fd
func (p pinger) OpenConn() (int, error) {
	//Create a raw socket to read icmp packets
	fd, err := p.syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	if err != nil {
		return 0, err
	}
	//Set the option to receive the kernel timestamp from each received message
	if err := p.syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_TIMESTAMP, 1); err != nil {
		return 0, err
	}
	//Increase the socket buffer
	if err := p.syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, 1024*1024); err != nil {
		return 0, err
	}
	//Configure to send IP Header together with the icmp message
	if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
		return 0, err
	}

	//Set timeout while reading from socket
	var tv syscall.Timeval
	tv.Sec = 1 /* 1 sec timeout */
	tv.Usec = 0
	if err := syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv); err != nil {
		return 0, err
	}
	//Listen on all interfaces
	var addr syscall.Sockaddr = &syscall.SockaddrInet4{
		Port: 0,
		Addr: [4]byte{0, 0, 0, 0},
	}
	if err := p.syscall.Bind(fd, addr); err != nil {
		return 0, err
	}
	return fd, nil
}

func (p pinger) ping(gpid int, fd int, in <-chan goping.SeqRequest, out chan<- goping.RawResponse, done chan<- struct{}) {
	var echo icmp.Echo
	var icmpmsg icmp.Message = icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
	}
	var iph ipv4.Header = ipv4.Header{
		Version:  4,
		Len:      20,
		Protocol: 1, // ICMP
	}
	var ip net.IP
	var icmpb, ipv4b []byte
	var buffer bytes.Buffer
	var tv syscall.Timeval

	var address = make(map[string]net.IP)
	var addressError = make(map[string]error)

	for r := range in {
		//Resolve HostName
		var err error
		if _, ok := address[r.Req.Host]; !ok {
			addr, err := net.ResolveIPAddr("ip4", r.Req.Host)
			if err != nil {
				addressError[r.Req.Host] = err
				address[r.Req.Host] = net.IPv4(0, 0, 0, 0)
			} else {
				address[r.Req.Host] = addr.IP.To4()
			}
		}
		if addressError[r.Req.Host] != nil {
			out <- goping.RawResponse{Seq: r.Seq, Err: errors.New("Could not resolve address"), RTT: math.NaN()}
			continue
		}
		//Create the target address to use in the SendTo socket method
		ip = address[r.Req.Host]
		var to syscall.SockaddrInet4
		to.Port = 0
		to.Addr[0], to.Addr[1], to.Addr[2], to.Addr[3] = ip[0], ip[1], ip[2], ip[3]

		//Built the Data to be send
		buffer.Reset()
		syscall.Gettimeofday(&tv)
		binary.Write(&buffer, binary.LittleEndian, &tv)
		echo.Data = buffer.Bytes()
		//Get the ICMP echo Identifier
		echo.ID = gpid
		//Get the ICMP echo sequence number
		echo.Seq = r.Seq
		//Build the ICMP echo body
		icmpmsg.Body = &echo
		//Build bytes of the ICMP echo
		if icmpb, err = icmpmsg.Marshal(nil); err != nil {
			out <- goping.RawResponse{Seq: r.Seq, Err: errors.New("Could not marshall ICMP Echo"), RTT: math.NaN()}
			continue
		}
		//Builds the ip header
		iph.TOS = int(r.Req.Config.TOS)
		iph.TotalLen = 20 + len(icmpb) // 20 bytes for IP, len(wb) for ICMP
		iph.TTL = int(r.Req.Config.TTL)
		iph.Dst = ip
		//Pack IP Header
		if ipv4b, err = iph.Marshal(); err != nil {
			out <- goping.RawResponse{Seq: r.Seq, Err: errors.New("Could not marshall IP Header"), RTT: math.NaN()}
			continue
		}
		//Sending the packet through the network
		if err = p.syscall.Sendto(fd, append(ipv4b, icmpb...), 0, &to); err != nil {
			out <- goping.RawResponse{Seq: r.Seq, Err: errors.New("Could not Send Ping over the socket"), RTT: math.NaN()}
			continue
		}
	}
	go func() {
		done <- struct{}{}
	}()
}

func (p pinger) pong(gpid int, fd int, out chan<- goping.RawResponse, donein <-chan struct{}, done chan<- struct{}) {
	//Buffer to receive the ping packet
	buf := make([]byte, 1024)
	//Buffer to receive the control message
	oob := make([]byte, 64)
	//Buffer to read oob
	var bbuf bytes.Buffer
	//Variable to get timestamp from oob
	var tv syscall.Timeval
	//Infinite loop to wait for messages
	for {
		//If doneIn is closed then close done channel and exit function
		select {
		case <-donein:
			close(done)
			if err := p.syscall.Close(fd); err != nil {
				fmt.Printf("Error calling syscall.Close %v\n", err)
			}
			return
		default:
		}

		//Receives a message from the socket sent by the kernel
		_, oobn, _, from, err := p.syscall.Recvmsg(fd, buf, oob, 0)
		if err != nil {
			//fmt.Printf("Error reading recvmsg %v\n", err)
			//Error reading packaging
			continue
		}
		var endTime = time.Now()

		//Finds the pid and seq value.
		var pid, seq int
		switch buf[20] {
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
		//Blocks processing if pid ! gpid or if packet is not an EchoRequest.
		if !(pid == gpid && buf[20] != 8) {
			continue
		}
		//Parses the Control Message to find the SO_TIMESTAMP value
		cmsgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
		if err != nil {
			//Error parsing socket control message
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
				endTime = time.Unix(tv.Unix())
			}
		}
		tv.Sec = 0
		tv.Usec = 0
		tvsz := unsafe.Sizeof(&tv)
		bbuf.Write(buf[28 : 28+tvsz*2])
		binary.Read(&bbuf, binary.LittleEndian, &tv)
		bbuf.Reset()
		var startTime = time.Unix(tv.Unix())

		//Get peer address
		peer := net.IPv4(
			from.(*syscall.SockaddrInet4).Addr[0],
			from.(*syscall.SockaddrInet4).Addr[1],
			from.(*syscall.SockaddrInet4).Addr[2],
			from.(*syscall.SockaddrInet4).Addr[3],
		)
		//GoRoutine that sends the raw response to channel out
		go func(seq int, msg []byte, peer net.IP, rtt float64) {
			out <- goping.RawResponse{Seq: seq, ICMPMessage: msg, Peer: peer, RTT: rtt}
		}(seq, buf[:40], peer, float64(endTime.Sub(startTime).Nanoseconds())/1e6)
	}
}

/*syscallWrapperInterface wraps syscall calls */
type syscallWrapperInterface interface {
	Socket(domain, typ, proto int) (fd int, err error)
	SetsockoptInt(fd, level, opt int, value int) (err error)
	Bind(fd int, sa syscall.Sockaddr) (err error)
	Sendto(fd int, p []byte, flags int, to syscall.Sockaddr) (err error)
	Recvmsg(fd int, p, oob []byte, flags int) (n, oobn int, recvflags int, from syscall.Sockaddr, err error)
	Close(fd int) (err error)
}

/*syscallWrapper is a type that implements syscallWrapperinterface */
type syscallWrapper struct{}

func (c syscallWrapper) Socket(domain, typ, proto int) (fd int, err error) {
	fd, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	return
}
func (c syscallWrapper) SetsockoptInt(fd, level, opt int, value int) (err error) {
	err = syscall.SetsockoptInt(fd, level, opt, value)
	return
}
func (c syscallWrapper) Bind(fd int, sa syscall.Sockaddr) (err error) {
	err = syscall.Bind(fd, sa)
	return
}
func (c syscallWrapper) Sendto(fd int, p []byte, flags int, to syscall.Sockaddr) (err error) {
	err = syscall.Sendto(fd, p, flags, to)
	return
}
func (c syscallWrapper) Recvmsg(fd int, p, oob []byte, flags int) (n, oobn int, recvflags int, from syscall.Sockaddr, err error) {
	n, oobn, recvflags, from, err = syscall.Recvmsg(fd, p, oob, flags)
	return
}
func (c syscallWrapper) Close(fd int) (err error) {
	err = syscall.Close(fd)
	return
}
