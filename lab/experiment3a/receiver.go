package ggping

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
	"syscall"
	"time"
)

//Verify if the packet is a reply from this process PID
//Get the Packet Sequence Number
//Get the time when the packet arrived in the Kernel using the SO_TIMESTAMP control message
//Send the timestamp through the channel inside pingarray using the seq field as index
func receiver(pingarray []chan time.Time) {

	//Finds out the process pid
	mypid := os.Getpid()

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
			log.Fatal(err)
		} else {

			//Continue if id is different from mypid and if icmp type is not ICMPTypeEchoReply
			if !(int(uint16(buf[24])<<8|uint16(buf[25])) == mypid && buf[20] == 0) {
				continue
			}

			//Sequence ping found
			seq := int(uint16(buf[26])<<8 | uint16(buf[27]))

			//Variable to hold  the timestamp
			var t time.Time

			//Parse the received control message until the oobn size
			cmsgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
			if err != nil {
				log.Fatal(os.NewSyscallError("parse socket control message", err))
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
					t = time.Unix(tv.Unix())
				}
			}

			//Send the timestamp through the pingarray[seq] channel
			pingarray[seq] <- t
		}
	}
}
