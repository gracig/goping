package goping

import (
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	//IPV4 is the ip version to choose in the Ping.IPVersion field
	IPV4 = 4
	//ICMP is the IP Protocol to choose in the Ping.IPProto field
	ICMP = 1
	//MAXSEQUENCE is the max number of the ICMP sequence field
	MAXSEQUENCE = 65536
)

//Ping is the ICMP Request to be done
type Ping struct {
	/*Control */
	Host      string
	Timeout   uint
	Interval  uint
	Count     uint
	Data      map[string]string
	IPVersion uint
	IPProto   uint

	/* Protocol */
	IP          *net.IPAddr
	IPV4Header  ipv4.Header
	IPV6Header  ipv6.Header
	ICMPMessage icmp.Message

	/*Statistics*/
	Sent   float64
	Recv   float64
	Failed float64
	AvgRtt float64
	MaxRtt float64
	MinRtt float64

	pinger Pinger
}

//Pong is the response for each Ping.
type Pong struct {
	Seq  uint
	Size uint
	RTT  float64

	Err error

	IPV4Header  ipv4.Header
	IPV6Header  ipv6.Header
	ICMPMessage icmp.Message
}

//Response is the object passed to the use everytime a pong is received. When is the last response, Done = true
type Response struct {
	Ping
	Pong
	Done bool
}

//GoPinger is the interface that
type Pinger interface {
	//Ping sends the Ping struct to the mainLoop through the chPing channel
	Send(p Ping)

	//Pong receives responses from the mainLoop
	PongChan() <-chan Response

	//pong used by the mainLoop to send the responses
	pongChan() chan<- Response
}

//Implements the context interface
type pinger struct {
	pongch chan Response
}

func (p pinger) Send(ping Ping) {
	ping.pinger = p
	chPing <- ping
}

func (p pinger) PongChan() <-chan Response {
	return p.pongch
}
func (p pinger) pongChan() chan<- Response {
	return p.pongch
}

var (
	chPing   = make(chan Ping)     //Receives ICMP requests
	chPong   = make(chan Pong)     //Receives ICMP Responses
	chPause  = make(chan struct{}) //Receives Pause commands
	chResume = make(chan struct{}) //Receives Resume commands
	ctrl     = struct {            //Status of mainLoop. Also mutex for status change synchronization
		paused  bool
		running bool
		sync.Mutex
	}{paused: false, running: false}
)

//New Creates a new Pinger.
func New() Pinger {
	return pinger{
		pongch: make(chan Response),
	}
}

/*
func addPing(ping Ping) error {

	var err error
		//Resolves the IP Address
		if ping.IPVersion == IPV4 {
			ping.IP, err = net.ResolveIPAddr("ip4", ping.Host)
			if err != nil {
				return fmt.Errorf("Could not resolve address: %s\n", ping.Host)
			}
		} else {
			return fmt.Errorf("Only IPVersion=IPV4 is supported right now!")
		}
	ping.ctx = ctx
	chPing <- ping
	return nil
}

*/
//Pause pauses the main loop forever.
//All ping requests and Pong responses are no longer processed . Even timeouts
//One should call Reume() to continue the main loop operation
func Pause() {
	ctrl.Lock()
	defer ctrl.Unlock()
	if !ctrl.paused {
		chPause <- struct{}{}
		ctrl.paused = true
	}
}

//Resume resumes the main loop, if it is paused.
func Resume() {
	ctrl.Lock()
	defer ctrl.Unlock()
	if ctrl.paused {
		chResume <- struct{}{}
		ctrl.paused = false
	}
}

//Run executes the main loop, exactly once.  It is called at init()
func Run() {
	ctrl.Lock()
	defer ctrl.Unlock()
	if !ctrl.running {
		ctrl.running = true
		go mainLoop()
	}
}

//mainLoop controls the schedule of ICMP requests (pings) and ICMP responses (pongs). And leads with timeouts
func mainLoop() {

	var (
		//Maps the ICMP sequence field with a Pong channel. It is set in chPing and read in chPong.
		recvchans = make([]chan Pong, MAXSEQUENCE, MAXSEQUENCE)
		//ctxPingCounter is a map used to control if response channel should be closed after  a Ping request is Done
		ctxPingCounter = make(map[<-chan Response]uint64)
		//totalPings counts the number of pings that have been sent from this engine
		totalPings uint64
	)

	//The main loop starts here
	for {
		select {
		case ping := <-chPing:
			//Incrementing context counter if first ping
			if ping.Sent == 0 {
				ctxPingCounter[ping.pinger.PongChan()]++
				//Validates ping object
				/*
					if ping.IPVersion == IPV4 {
						ping.IP, err = net.ResolveIPAddr("ip4", ping.Host)
						if err != nil {
							return fmt.Errorf("Could not resolve address: %s\n", ping.Host)
						}
					} else {
						return fmt.Errorf("Only IPVersion=IPV4 is supported right now!")
					}
				*/
			}
			//Sending the Done Response when number of ping.Sent is equals the number of ping.Count
			if uint(ping.Sent) >= ping.Count {
				ping.pinger.pongChan() <- Response{ping, Pong{}, true}
				//Closing the channel if no ping requests is left on the given context.
				ctxPingCounter[ping.pinger.PongChan()]--
				if ctxPingCounter[ping.pinger.PongChan()] <= 0 { //If counter is <= 0 then we should close the channel on context
					delete(ctxPingCounter, ping.pinger.PongChan()) //Deleting the map key.
					close(ping.pinger.pongChan())                  //Closing the channel on context
				}
			} else {
				//Incrementing ping counters
				totalPings++
				ping.Sent++
				//Creating ICMP sequence and Pong channel. associating them in recvchans
				sequence := totalPings % MAXSEQUENCE
				if recvchans[sequence] != nil {
					close(recvchans[sequence])
					recvchans[sequence] = nil
				}
				recvchans[sequence] = make(chan Pong, 1)
				//Create and send PingRequest

				//Wait for a response, timeout if necessary and waits interval after send itself to chping channel
				go func(sequence uint64, ping Ping, chrecv chan Pong) {
					var pong Pong
					wait := time.NewTimer(time.Millisecond * time.Duration(ping.Interval))
					tout := time.NewTimer(time.Millisecond * time.Duration(ping.Timeout))
					select {
					case <-tout.C:
						//Create a timeout Pong
						pong = Pong{
							Seq: uint(sequence),
							RTT: math.NaN(),
							Err: fmt.Errorf("Timeout"),
						}
					case pong = <-chrecv: //Received Pong
					}
					ping.pinger.pongChan() <- Response{ping, pong, false} //Send Response to context channel. Done is false
					<-wait.C                                              //Waits for the interval
					chPing <- ping                                        //Continue Ping operation
				}(sequence, ping, recvchans[sequence])
			}
		case pong := <-chPong: //Receiving ICMP Response
			if recvchans[pong.Seq] != nil {
				recvchans[pong.Seq] <- pong
				close(recvchans[pong.Seq])
				recvchans[pong.Seq] = nil
			}
		case <-chPause:
			<-chResume //Blocks until a resume command is received
		}
	}
}
