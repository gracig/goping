package icmpv4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"github.com/gracig/goping"
)

func New() goping.Pinger {
	return &pinger{}
}

//Pinger is the type the implements goping.Pinger interface
type pinger struct {
}

//Start is the implementation of the method goping.Pinger.Start
func (p pinger) Start(pid int) (ping chan<- goping.SeqRequest, pong <-chan goping.RawResponse, done <-chan struct{}, err error) {
	//Initialize the channels used in the select stage
	input, output, doneInput, doneOutput := make(chan goping.SeqRequest), make(chan goping.RawResponse), make(chan struct{}), make(chan struct{})
	var c net.PacketConn
	//Opens the connection
	c, err = net.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Printf("ListenPacket failed: %v\n", err)
		return
	}
	go p.pong(pid, c, output, doneInput, doneOutput)
	go p.ping(pid, c, input, output, doneInput)

	ping, pong, done = input, output, doneOutput
	return
}

func (p pinger) ping(gpid int, c net.PacketConn, input <-chan goping.SeqRequest, output chan<- goping.RawResponse, doneInput chan struct{}) {
	var echo icmp.Echo
	var icmpmsg icmp.Message = icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
	}
	var address = make(map[string]*net.IPAddr)
	var addressError = make(map[string]error)

	var ip *net.IPAddr
	var icmpb []byte
	var buffer bytes.Buffer

	var nanob = make([]byte, 16, 16)
	for r := range input {
		//Resolve HostName
		var err error
		if _, ok := address[r.Req.Host]; !ok {
			addr, err := net.ResolveIPAddr("ip4", r.Req.Host)
			if err != nil {
				addressError[r.Req.Host] = err
				address[r.Req.Host] = nil
			} else {
				address[r.Req.Host] = addr
			}
		}
		if addressError[r.Req.Host] != nil {
			output <- goping.RawResponse{Seq: r.Seq, Err: errors.New("Could not resolve address"), RTT: math.NaN()}
			continue
		}

		//Create the target address to use in the SendTo socket method
		ip = address[r.Req.Host]

		//Built the Data to be send
		buffer.Reset()
		nano := time.Now().UnixNano()
		binary.PutVarint(nanob, nano)
		echo.Data = nanob

		//Get the ICMP echo Identifier
		echo.ID = gpid
		//Get the ICMP echo sequence number
		echo.Seq = r.Seq
		//Build the ICMP echo body
		icmpmsg.Body = &echo
		//Build bytes of the ICMP echo
		if icmpb, err = icmpmsg.Marshal(nil); err != nil {
			output <- goping.RawResponse{Seq: r.Seq, Err: errors.New("Could not marshall ICMP Echo"), RTT: math.NaN()}
			continue
		}
		if _, err := c.WriteTo(icmpb, ip); err != nil {
			output <- goping.RawResponse{Seq: r.Seq, Err: errors.New("Could send icmp message"), RTT: math.NaN()}
		}
	}
}
func (p pinger) pong(gpid int, c net.PacketConn, output chan<- goping.RawResponse, doneInput, doneOutput chan struct{}) {
	//Buffer to receive the ping packet
	buf := make([]byte, 1024)

	for {

		var start = time.Time{}
		var bbuf bytes.Buffer

		select {
		case <-doneInput:
			close(doneOutput)
			if err := c.Close(); err != nil {
				fmt.Printf("Error calling PacketConn.Close %v\n", err)
			}
			return
		default:
		}

		c.SetReadDeadline(time.Now().Add(time.Second))

		n, peer, err := c.ReadFrom(buf)

		if err != nil {
			continue
		}

		//Finds the pid and seq value.
		var pid, seq int
		switch buf[0] {
		case 0:
			//Received an Echo Reply
			pid = int(uint16(buf[4])<<8 | uint16(buf[5]))
			seq = int(uint16(buf[6])<<8 | uint16(buf[7]))

			bbuf.Reset()
			bbuf.Write(buf[8:n])

			if nano, err := binary.ReadVarint(&bbuf); err != nil {
			} else {
				start.Add(time.Duration(nano))
			}
			if pid != gpid {
				continue
			}
			end := time.Now()
			go func(seq int, msg []byte, peer net.IP, rtt float64) {
				output <- goping.RawResponse{Seq: seq, ICMPMessage: msg, Peer: peer, RTT: rtt}
			}(seq, buf[:40], net.ParseIP(peer.String()), float64(end.Sub(start).Nanoseconds())/1e6)

		default:
			//Received an error message
			pid = int(uint16(buf[32])<<8 | uint16(buf[33]))
			seq = int(uint16(buf[34])<<8 | uint16(buf[35]))
			//emsg = msg[28:] //20+28+4
			if pid != gpid {
				continue
			}
			go func(seq int, msg []byte, peer net.IP, rtt float64) {
				output <- goping.RawResponse{Seq: seq, ICMPMessage: msg, Peer: peer, RTT: rtt, Err: errors.New("Error receiving message")}
			}(seq, buf[:40], net.ParseIP(peer.String()), math.NaN())
		}
	}
}
