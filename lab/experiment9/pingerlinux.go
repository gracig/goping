package ggping

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

		seqPongChanArray: make([]chan *Pong, maxseq),
		seqStartArray:    make([]time.Time, maxseq),
		mypid:            os.Getpid(),
		done:             make(chan struct{}),
	}
}

//Send Icmp Packets over the network
//Receives request from the Ping Channel
//Register the new sequence in the pingarray that the receiver will send the timestamp
//Send Icmp Packet through the network
//Puts the Request back on the pong channel. The caller will verify the response and timeout
func (p *linuxPinger) Start(ping chan *Ping, pong chan *Ping) {

	//Maintains a sequence number
	var seq int

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
	for pi := range ping {
		debug.Println("Received ping", pi)

		//Increment the sequence number and assigns to pi.Seq
		seq = (seq + 1) % maxseq
		debug.Printf("Next sequence will be %v\n", seq)

		//Get the marshalled bytes
		pkt, err := pi.Marshall(p.mypid, seq, true)
		ddebug.Printf("The marshalled packet [%v] \n%v\n", seq, pkt)
		if err != nil {
			log.Fatal(err)
		}

		//Builds the target SockaddrInet4
		ipb := pi.toaddr.IP.To4()
		to := syscall.SockaddrInet4{
			Port: 0,
			Addr: [4]byte{
				ipb[0],
				ipb[1],
				ipb[2],
				ipb[3],
			},
		}

		//Stores the PongChan in a array indexed by the icmp sequence number
		//It will be used by the listener to send the pong back
		debug.Println("Setting the pongchan channel on the seqPongChan aray", seq)
		p.seqPongChanArray[seq] = pi.pongchan

		//Setting the time before send the packet
		//This information will be read by the listener in order to build the Pong object
		debug.Println("Setting the Start time and sending through the network", seq)
		p.seqStartArray[seq] = time.Now()

		//Sending the packet through the network
		if err = syscall.Sendto(fd, pkt, 0, &to); err != nil {
			pi.err = fmt.Errorf("Could not send packet over socket: %v", err)
		}

		//This  permits other goroutines to run and avoid saturate the socket with too
		//many simultaneous Echo Requests
		debug.Printf("Sleeping for %v [%v]\n", seq, p.Sweep)
		time.Sleep(p.Sweep)
		pi.when = p.seqStartArray[seq]

		//Returns the ping through the pong channel. The receiver will send a timestamp if a packet arrived for this sequence
		debug.Printf("Returning Ping %v\n", seq)
		pong <- pi
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
		if _, oobn, _, _, err := syscall.Recvmsg(fd, buf, oob, 0); err != nil {
			log.Printf("Error reading icmp packet from the socket: %v", err)
			return
		} else {

			//Continue if id is different from mypid and if icmp type is not ICMPTypeEchoReply
			if !(int(uint16(buf[24])<<8|uint16(buf[25])) == p.mypid && buf[20] == 0) {
				ddebug.Printf("IGNORED packet %v\n", buf)
				continue
			}

			ddebug.Printf("ACCEPTED packet %v\n", buf)
			//Extracting the sequence field from the packet
			seq := int(uint16(buf[26])<<8 | uint16(buf[27]))
			debug.Printf("Received Pong Sequence %v\n", seq)

			//Build the Pong Object to be sent. The Echo End is filled with time.Now in case the timestamp
			//from SO_TIMESTAMP fails
			pong := Pong{
				EchoStart: p.seqStartArray[seq],
				EchoEnd:   time.Now(),
				Sequence:  seq,
			}
			debug.Printf("Pong Object created Sequence %v %v\n", seq, pong)

			//Parse the received control message until the oobn size
			cmsgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
			if err != nil {
				pong.err = os.NewSyscallError("parse socket control message", err)
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
			p.seqPongChanArray[seq] <- &pong
			ddebug.Println("Sent pong reference to the ping.pongchan reference ", p.seqPongChanArray[seq], seq)
		}
	}
}
