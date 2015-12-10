package ggping

import (
	"bytes"
	"encoding/binary"
	"log"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type linuxPinger struct {
	Sweep                 time.Duration
	fd                    int //The write file descriptor
	stopListen            chan struct{}
	seqEchoReplyChanArray []chan EchoReply //Array to send EchoReply to the caller
	done                  chan struct{}    //Channel to stop the pongreceiver
	mu                    sync.Mutex
}

func newLinuxPinger(sweep time.Duration) linuxPinger {
	if sweep == 0 {
		debug.Println("Setting sweep to 1 nanosecond")
		sweep = 0 * time.Nanosecond
	}
	return linuxPinger{
		Sweep: sweep,
		seqEchoReplyChanArray: make([]chan EchoReply, g_SEQUENCE.max),
	}
}

//Send Icmp Packets over the network
//Receives request from the Ping Channel
//Send Icmp Packet through the network
//Puts the Request back on the pong channel. The caller will verify the response and timeout
func (p *linuxPinger) Start(ping chan EchoRequest) {

	//Create a raw socket to read icmp packets
	debug.Printf("Creating the socket to send pings\n")
	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	p.fd = fd

	//Tells the socket we will build the ip header
	debug.Printf("Set the options to warn the kernel that the program will build the pi header\n")
	if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
		log.Fatal("Could not set sock opt syscall HDRINCL")
	}

	var pingcounter int64

	//Starts the pong receiver in a goroutine
	go p.pongReceiver()

	//Process the ping requests
	//1) should get the pongchan channel from the received Ping pongchan array, using the pi.sent-1 as the array index
	//2) Use local logic to make the ping. If anything goes wrong should set the err field of the received Ping object
	//3) Should always return the Ping object without block waitng for a reply. The reply will be checked later through the pongchan channel
	for echo := range ping {
		//Count the number of pings
		pingcounter++

		//Ping
		p.ping(&echo)

		//This  permits other goroutines to run and  control the smoothness of te pings over the devices and the network
		debug.Printf("Sleeping for %v\n", p.Sweep)
		time.Sleep(p.Sweep)
	}
	info.Printf("Number of pings sent %v\n", pingcounter)
	p.done <- struct{}{}
}
func (p *linuxPinger) ping(echo *EchoRequest) {

	var when time.Time = time.Now()

	//Get the next sequence value
	sequence := int(g_SEQUENCE.next())

	//Build the Icmp Message
	icmpmsg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			Data: make([]byte, echo.Size),
			ID:   getId(),
			Seq:  int(sequence),
		},
	}

	//Marshall the icmp packet
	if icmpb, err := icmpmsg.Marshal(nil); err != nil {
		log.Printf("Could not Marshall ICMP Message. Echo:(%v) Error:(%v)\n", echo, err)
	} else {
		//Build the ip header
		iph := ipv4.Header{
			Version:  4,
			Len:      20,
			TOS:      int(echo.TOS),
			TotalLen: 20 + len(icmpb), // 20 bytes for IP, len(wb) for ICMP
			TTL:      int(echo.TTL),
			Protocol: 1, // ICMP
			Dst:      echo.To.IP,
		}

		if iphb, err := iph.Marshal(); err != nil {
			log.Printf("Could not Marshall IP Header. Echo:(%v) Error:(%v)\n", echo, err)
		} else {

			//Storing the pongchannel in the pongchanarray indexed by the sequence number
			//the pongchan is used to send the echoreply back (EchoReply object)
			p.seqEchoReplyChanArray[sequence] = echo.chreply

			//Get the marshalled bytes
			pkt := append(iphb, icmpb...)

			//Getting the Target Socket Address
			ipb := echo.To.IP.To4()
			addrInet4 := syscall.SockaddrInet4{
				Port: 0,
				Addr: [4]byte{
					ipb[0],
					ipb[1],
					ipb[2],
					ipb[3],
				},
			}

			//This information will be read by the listener in order to build the EchoReply object
			when = time.Now()

			//Sending the packet through the network
			if err := syscall.Sendto(p.fd, pkt, 0, &addrInet4); err != nil {
				log.Printf("Could not isent packet over the socket. Echo:(%v) Error:(%v)\n", echo, err)
			}
		}
	}

	//Send the time the ping was generated
	echo.WhenSent = when
	echo.chwhen <- when
	g_PKT_SENT.next()

}

func (p *linuxPinger) checkTimeout() (chtimeout chan *EchoRequest, chdone chan struct{}) {
	chtimeout, chdone = make(chan *EchoRequest), make(chan struct{})
	var timeoutslice []*EchoRequest
	ticker := time.NewTicker(time.Millisecond * 1000)
	go func() {
		for {
			select {
			case echo := <-chtimeout:
				timeoutslice = append(timeoutslice, echo)
			case <-chdone:
				for _, echo := range timeoutslice {
					echo.chtimeout <- time.Now()
				}
				timeoutslice = nil
				info.Printf("Closing checkTimeout Loop")
				return
			case <-ticker.C:
			}
		}
	}()
	return
}

//Verify if the packet is a reply from this process PID
//Get the Packet Sequence Number
//Get the time when the packet arrived in the Kernel using the SO_TIMESTAMP control message
//Create a EchoReply object and send it through the channel inside pingarray using the seq field as index
func (p *linuxPinger) pongReceiver() {

	//Create a raw socket to read icmp packets
	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)

	//Set the option to receive the kernel timestamp from each received message
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_TIMESTAMP, 1); err != nil {
		log.Fatal("Could not set sock opt syscall")
	}

	//	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, 1024*1024*1024); err != nil {
	//		log.Fatal("Could not set sock opt syscall Increase buffer")
	//	}

	//Create an ip address to listen for
	var addr syscall.Sockaddr = &syscall.SockaddrInet4{
		Port: 0,
		Addr: [4]byte{0, 0, 0, 0},
	}

	//Bind the created socket with the address to listen for
	if err := syscall.Bind(fd, addr); err != nil {
		log.Fatal("Could not bind")
	}

	//Preparing the syscall.Select
	rfds := &syscall.FdSet{}
	timeout := &syscall.Timeval{}
	timeout.Sec, timeout.Usec = 1, 0

	endloop := make(chan struct{})

	//The main for loop
	for {

		select {
		case <-p.done:
			go func() {
				info.Println("Waiting 10 seconds before leave ")
				time.Sleep(10 * time.Second)
				endloop <- struct{}{}
			}()
		case <-endloop:
			info.Printf("End Pong Receiver \n")
			return
		default:
			//Verify if socket has received icmp messages
			FD_ZERO(rfds)
			FD_SET(rfds, fd)
			_, err := syscall.Select(fd+1, rfds, nil, nil, timeout)
			if err != nil {
				log.Fatalln(err)
			}

			//Verify if we have data
			if FD_ISSET(rfds, fd) {
				//Process the Icmp Message
				p.readFromSocket(fd)
			}
			//Check if we should close thie for loop
		}
	}
}

func (p *linuxPinger) readFromSocket(fd int) {

	//Buffer to receive the ping packet
	buf := make([]byte, 1024)

	//Buffer to receive the control message
	oob := make([]byte, 64)

	//Receives a message from the socket sent by the kernel
	//Continue on error
	pktsz, oobn, _, _, err := syscall.Recvmsg(fd, buf, oob, 0)
	if err != nil {
		severe.Printf("Error reading icmp packet from the socket: %v\n", err)
		return
	}
	g_PKT_RECEIVED.next()

	//Extracting icmp message from thr packet
	msg := buf[20:pktsz]

	//Find the first 8 bytes from the Echo Request message
	emsg := msg[:8]

	//Finds if it was an error message, changes emsg accordingly
	//The default section ignores the message and tries to read other packets
	switch msg[0] {
	case 0:
	case 8:
		//Ignores EchoRequest packets from localhost
		return
	default:
		emsg = msg[28:]
	}

	//Extracts id and seq
	pid := uint8(emsg[4])
	seq := uint32(emsg[5])<<16 | uint32(emsg[6])<<8 | uint32(emsg[7])

	//Ignores processing if id is different from mypid
	if pid != g_IDENTIFIER {
		return
	}

	//Build the Echo Reply Object
	reply := EchoReply{
		WhenRecv: time.Now(),
		Type:     msg[0],
		Code:     msg[1],
	}

	//Parse the received control message until the oobn size
	cmsgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		log.Printf("Could not parse socket control message to retrieve timestamp from socket")
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
			//Found Timestamp. Using binary package to read from
			var tv syscall.Timeval
			binary.Read(bytes.NewBuffer(m.Data), binary.LittleEndian, &tv)
			reply.WhenRecv = time.Unix(tv.Unix())
		}
	}

	//Send the pong object  back to the caller
	p.seqEchoReplyChanArray[seq] <- reply

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
