package goping

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"
	"time"
)

type linuxPinger struct {
	Sweep time.Duration

	mypid            int
	stopListen       chan struct{}
	seqPongChanArray []chan *Pong  //Array to send Pong to the caller
	seqStartArray    []time.Time   //Array to receive the Time when the ping was sent. Used by the listener to build the Pong object
	done             chan struct{} //Channel to stop the pongreceiver
	mu               sync.Mutex
}

func newLinuxPinger(sweep time.Duration) linuxPinger {
	return linuxPinger{
		Sweep: sweep,

		seqPongChanArray: make([]chan *Pong, g_SEQUENCE.max),
		seqStartArray:    make([]time.Time, g_SEQUENCE.max),
		mypid:            os.Getpid(),
		done:             make(chan struct{}),
	}
}

//Send Icmp Packets over the network
//Receives request from the Ping Channel
//Send Icmp Packet through the network
//Puts the Request back on the pong channel. The caller will verify the response and timeout
func (p *linuxPinger) Start(ping chan *EchoRequest, pong chan *EchoRequest) {
	defer close(pong)

	//Create a raw socket to read icmp packets
	debug.Printf("Creating the socket to send pings\n")
	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)

	//Tells the socket we will build the ip header
	debug.Printf("Set the options to warn the kernel that the program will build the pi header\n")
	if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
		log.Fatal("Could not set sock opt syscall HDRINCL")
	}

	//Starts the pong receiver in a goroutine
	debug.Printf("Starts the pong receiver in a go routine")
	go p.pongReceiver()

	//Process the ping requests
	//1) should get the pongchan channel from the received Ping pongchan array, using the pi.sent-1 as the array index
	//2) Use local logic to make the ping. If anything goes wrong should set the err field of the received Ping object
	//3) Should always return the Ping object without block waitng for a reply. The reply will be checked later through the pongchan channel
	for echo := range ping {
		debug.Printf("Received Echo Request pongchan:[%v] [%v]\n", echo.pongchan, echo.when)

		//Storing the pongchannel in the pongchanarray indexed by the sequence number
		//the pongchan is used to send the echoreply back (Pong object)
		p.seqPongChanArray[echo.sequence] = echo.pongchan

		//Get the marshalled bytes
		pkt := append(echo.bipv4, echo.bicmp...)
		ddebug.Printf("The marshalled packet [%v] \n%v\n", echo.sequence, pkt)

		//Builds the target SockaddrInet4
		ipb := echo.to.IP.To4()
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
		//This information will be read by the listener in order to build the Pong object
		debug.Println("Setting the Start time and sending through the network", echo.sequence)
		p.seqStartArray[echo.sequence] = time.Now()

		//Sending the packet through the network
		if err := syscall.Sendto(fd, pkt, 0, &to); err != nil {
			echo.err = fmt.Errorf("Could not send packet over socket: %v", err)
		}

		//This  permits other goroutines to run and avoid saturate the socket with too many simultaneous Echo Requests
		debug.Printf("Sleeping for %v [%v]\n", echo.sequence, p.Sweep)
		time.Sleep(p.Sweep)
		echo.when = p.seqStartArray[echo.sequence]

		//Returns the ping through the pong channel. The receiver will send a timestamp if a packet arrived for this sequence
		debug.Printf("Returning Echo Request index: [%v] sequence:[%v] pongchan:[%v]\n", echo.index, echo.sequence, echo.pongchan)
		pong <- echo
		debug.Printf("Returned Echo Request index: [%v] sequence:[%v] pongchan:[%v]\n", echo.index, echo.sequence, echo.pongchan)
	}
}

//Verify if the packet is a reply from this process PID
//Get the Packet Sequence Number
//Get the time when the packet arrived in the Kernel using the SO_TIMESTAMP control message
//Create a Pong object and send it through the channel inside pingarray using the seq field as index
func (p *linuxPinger) pongReceiver() {

	//Create a raw socket to read icmp packets
	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)

	//Set the option to receive the kernel timestamp from each received message
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_TIMESTAMP, 1); err != nil {
		log.Fatal("Could not set sock opt syscall")
	}

	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, 1024*1024*1024); err != nil {
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

	for {
		//Buffer to receive the ping packet
		buf := make([]byte, 1024)

		//Buffer to receive the control message
		oob := make([]byte, 64)

		//Receives a message from the socket sent by the kernel
		//Continue on error
		pktsz, oobn, _, peer, err := syscall.Recvmsg(fd, buf, oob, 0)
		if err != nil {
			severe.Printf("Error reading icmp packet from the socket: %v\n", err)
			continue
		}
		//Exracting ip header from the packet
		iph := buf[:20]
		//Extracting icmp message from thr packet
		msg := buf[20:pktsz]

		//Find the first 8 bytes from the Echo Request message
		emsg := msg[:8]

		//Finds if it was an error message, changes emsg accordingly
		//The default section ignores the message and tries to read other packets
		var icmperror error
		switch msg[0] {
		case 0:
		case 8:
			continue
		default:
			emsg = msg[28:]
			icmperror = fmt.Errorf("There was an error with code %v", msg[0])
		}

		//Extracts id and seq
		pid := int(uint16(emsg[4])<<8 | uint16(emsg[5]))
		seq := int(uint16(emsg[6])<<8 | uint16(emsg[7]))

		//Ignores processing if id is different from mypid
		if pid != p.mypid {
			continue
		}

		//Build the Pong Object to be sent. The Echo End is filled with time.Now in case the timestamp
		//from SO_TIMESTAMP fails
		pong := Pong{
			EchoStart: p.seqStartArray[seq],
			EchoEnd:   time.Now(),
			iph:       iph,
			imh:       msg,
			Err:       icmperror,
			peer:      peer.(*syscall.SockaddrInet4).Addr[:],
		}

		//Parse the received control message until the oobn size
		cmsgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
		if err != nil {
			pong.Err = os.NewSyscallError("parse socket control message", err)
		}
		debug.Println("Controle Message received from socket was parsed")

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
				debug.Println("SO_TIMESTAMP message found")
				var tv syscall.Timeval
				binary.Read(bytes.NewBuffer(m.Data), binary.LittleEndian, &tv)
				pong.EchoEnd = time.Unix(tv.Unix())
				debug.Println("Pong EchoEnd changed ", pong)
			}
		}

		//Send the pong object  back to the caller
		debug.Println("Sending pong reference to the ping.pongchan reference ", p.seqPongChanArray[seq], seq)
		go func() {
			p.seqPongChanArray[seq] <- &pong
			debug.Println("Sent pong reference to the ping.pongchan reference ", p.seqPongChanArray[seq], seq)
		}()
	}
}
