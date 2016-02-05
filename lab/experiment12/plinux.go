package goping

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type icmpLinux struct {
	chping chan *pinger
	wginit sync.WaitGroup
}

//Implements osping
func (this *icmpLinux) init() {
	this.chping = make(chan *pinger)
	this.wginit.Add(2)
	go this.icmpReceiver()
	go this.icmpSender()
	this.wginit.Wait()
}

//Implements osping
func (this *icmpLinux) ping(p *pinger) time.Time {
	this.chping <- p
	return <-p.chWhen
}

//Send icmp Requests over a raw socket
func (this *icmpLinux) icmpSender() {

	//Create a raw socket to read icmp packets
	debug.Printf("Creating the socket to send pings\n")
	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)

	//Increase the socket buffer
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_SNDBUF, 65000); err != nil {
		log.Fatal("Could not set sock opt syscall Increase buffer")
	}

	//Tells the socket we will build the ip header
	debug.Printf("Set the options to warn the kernel that the program will build the pi header\n")
	if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
		log.Fatal("Could not set sock opt syscall HDRINCL")
	}

	this.wginit.Done()

	//To reuse a icmp.Echo object and avoi memory leaks
	var err error
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

	for p := range this.chping {
		//Build the Icmp Message
		echo.Data = p.packetbytes
		echo.ID = g_IDENTIFIER
		echo.Seq = p.Seq
		icmpmsg.Body = &echo

		if icmpb, err = icmpmsg.Marshal(nil); err != nil {
			log.Printf("Could not Marshall ICMP Message. IP:(%v) Error:(%v)\n", p.Target, err)
			p.chWhen <- time.Now()
			continue
		}

		//Builds the ip header
		iph.TOS = int(p.Settings.TOS)
		iph.TotalLen = 20 + len(icmpb) // 20 bytes for IP, len(wb) for ICMP
		iph.TTL = int(p.Settings.TTL)
		iph.Src = p.Target.From.IP
		iph.Dst = p.Target.IP.IP

		//Pack IP Header
		if ipv4b, err = iph.Marshal(); err != nil {
			log.Printf("Could not Marshall IP Header. IP:(%v) Error:(%v)\n", p.Target, err)
			p.chWhen <- time.Now()
			continue
		}

		//Create the target address to use in the SendTo socket method
		ip = p.Target.IP.IP.To4()
		var to syscall.SockaddrInet4
		to.Port = 0
		to.Addr[0], to.Addr[1], to.Addr[2], to.Addr[3] = ip[0], ip[1], ip[2], ip[3]

		//This information will be read by the listener in order to build the EchoReply object
		when := time.Now()

		//Sending the packet through the network
		if err = syscall.Sendto(fd, append(ipv4b, icmpb...), 0, &to); err != nil {
			log.Fatal("Could not send packet over socket:", err)
		}
		p.chWhen <- when
	}
}

//Receives icmp responses from a raw socket. Get timestamp from the SO_TIMESTAMP flag
func (this *icmpLinux) icmpReceiver() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	//Create a raw socket to read icmp packets
	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)

	//Set the option to receive the kernel timestamp from each received message
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_TIMESTAMP, 1); err != nil {
		log.Fatal("Could not set sock opt syscall")
	}

	//Increase the socket buffer
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, 1024*1024); err != nil {
		log.Fatal("Could not set sock opt syscall Increase buffer")
	}

	//Create an ip address to listen for
	var addr syscall.Sockaddr = &syscall.SockaddrInet4{
		Port: 0,
		Addr: [4]byte{0, 0, 0, 0},
	}

	//Bind the created socket with the address to listen for
	if err := syscall.Bind(fd, addr); err != nil {
		log.Fatal("Could not bind")
	}

	this.wginit.Done()

	//Buffer to receive the ping packet
	buf := make([]byte, 1024)

	//Buffer to receive the control message
	oob := make([]byte, 64)

	//Variables used in the for loop
	var pid, seq int
	var tv syscall.Timeval
	var bbuf bytes.Buffer

	//Acual Ring Position
	ps := pstatus{r: g_PINGER_RING}

	for {
		//Receives a message from the socket sent by the kernel
		_, oobn, _, peer, err := syscall.Recvmsg(fd, buf, oob, 0)
		if err != nil {
			info.Printf("Error reading icmp packet from the socket: %v\n", err)
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
