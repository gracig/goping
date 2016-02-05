package goping

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var chping chan *Request
var checho chan rawIcmp

var identifier int = (os.Getpid() & 0xffff)
var seq int32 //The Next Sequence to be used

//Requests a Synchronous Ping (Internally channels are used)
func Ping(p *Request) (*Pong, error) {

	chpong := make(chan *Pong) //Channel that will be used to receive the Pong
	defer close(chpong)

	PingOnChan(p, chpong) //Calls the PingChan function and waits for the answer

	pong := <-chpong //Waits for the pong object

	return pong, pong.Err //Returns the pong object and the error if any
}
func PingOnChan(req *Request, c chan *Pong) {
	//Assign the return channel to the Request
	req.pong = c

	//Got the sequence number for the ping
	req.seq = int(atomic.AddInt32(&seq, 1))

	//Validate the  Ip on Request
	ip, err := net.ResolveIPAddr("ip4", req.To)
	if err != nil {
		go func() {
			req.pong <- &Pong{Request: req, Err: fmt.Errorf("Could not resolve Ip address for: %v", req.To)}
		}()
		return
	}
	//Assign the resolved IP address to the request
	req.ip = ip

	//Send request to process
	go func() {
		chping <- req
	}()
}

type Request struct {
	To      string     //The Ip or FQDN of the host to ping
	Timeout int        //Timeout to receive a pong (answer for the ping)
	pong    chan *Pong //The channel to receive the Pong Object
	seq     int        //The Ping sequence
	ip      *net.IPAddr
	rawicmp chan *rawIcmp //Channel to receive the raw ICMP
}

type Pong struct {
	Request *Request //The ping request
	Rtt     float64  //The  time elapsed between the ping sent over the wire and the arrived pong response
	Err     error
	Packet  *icmp.Echo
	Id      int       //The id of this ping. The PID will be used
	Seq     int       //Sequence of the ping created byt this api
	done    chan bool // Indicates we receive a response, either a Pong or a Timeout
}
type rawIcmp struct {
	when    time.Time
	size    int
	peer    net.Addr
	bytes   []byte
	message *icmp.Echo //The message after being parsed
	err     error
}

//Parse the message. Generate error if Id is differente from PID
func (r *rawIcmp) parseMessage() error {
	if message, err := icmp.ParseMessage(1, r.bytes[:r.size]); err != nil {
		return err
	} else {
		r.message = message.Body.(*icmp.Echo)
	}
	return nil
}

func init() {
	chping := make(chan *Request)

	//Opens the raw socket using the package golang/x/icmp from google
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Fatal("Could not open raw socket ip4:icmp: %v", err)
	}

	//Starts an infinite loop to read from raw socket icmp
	go func() {
		//Opens the raw socket using the package golang/x/icmp from google
		c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		if err != nil {
			log.Fatal("Could not open raw socket ip4:icmp: %v", err)
		}

		for {
			//Reads an ICMP Message from the Socket.
			ri := rawIcmp{bytes: make([]byte, 1500)}
			if ri.size, ri.peer, ri.err = c.ReadFrom(ri.bytes); ri.err != nil {
				log.Fatal("Could not read from socket: %v", ri.err)
			}

			//Tags the time when the message arrived. This will be used to calc RTT
			ri.when = time.Now()

			//Sends the Message to the checho channel
			go func(ri rawIcmp) {
				checho <- ri
			}(ri)
		}
	}()

	go func() {
		requests := make(map[int]chan<- *rawIcmp)
		for {
			select {
			case req := <-chping:
				wm := icmp.Message{
					Type: ipv4.ICMPTypeEcho, Code: 0,
					Body: &icmp.Echo{
						ID: identifier, Seq: req.seq,
						Data: []byte("HELLO-R-U-THERE####All ALL OTHER STUFF!"),
					},
				}
				if wb, err := wm.Marshal(nil); err != nil {
					go func() {
						req.pong <- &Pong{Request: req, Err: fmt.Errorf("Could not marshal icmp packet: [%v] %v", err, req)}
					}()
				} else {
					if _, err := c.WriteTo(wb, req.ip); err != nil {
						go func() {
							req.pong <- &Pong{Request: req, Err: fmt.Errorf("Could not write to socket: [%v] %v", err, req)}
						}()
					}
				}
				start := time.Now()
				requests[req.seq] = req.rawicmp
				go func(req *Request, start time.Time) {
					select {
					case raw := <-req.rawicmp: //Received icmp response
						req.pong <- &Pong{
							Request: req,
							Rtt:     float64(time.Since(start)-time.Since(raw.when)) * float64(time.Second/time.Millisecond),
							Packet:  raw.message,
						}
						delete(requests, req.seq)
					case <-time.After(time.Second * time.Duration(req.Timeout)): //Icmp Response Timed out
						req.pong <- &Pong{Request: req, Err: fmt.Errorf("Timed Out", err, req)}
						delete(requests, req.seq)
					}

				}(req, start)

			case raw := <-checho:
				if raw.message.ID == identifier {
					if requests[raw.message.Seq] != nil {
						requests[raw.message.Seq] <- &raw
					}
				}
			}
		}
	}()
}
