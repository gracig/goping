package goping

import (
	"fmt"
	"math"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	//IPV4 is the ip version to choose in the Ping.IPVersion field
	IP4 = 4
	//ICMP is the IP Protocol to choose in the Ping.IPProto field
	ProtoICMP = 1
	//MAXSEQUENCE is the max number of the ICMP sequence field
	MAXSEQUENCE = 65536
)

//Ping is the ICMP Request to be done
type Ping struct {
	/*Control */
	Host      string
	Timeout   int
	Interval  int
	Count     int
	PcktSize  int
	Data      map[string]string
	IPVersion int
	IPProto   int

	/* Protocol */
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
}

//Pong is the response for each Ping.
type Pong struct {
	//RTT is the Round Trip Time of the packet
	RTT float64

	//Err is generated when packets weren't send
	Err error

	/* Protocol */
	IPV4Header  ipv4.Header
	IPV6Header  ipv6.Header
	ICMPMessage icmp.Message
}

//Response is the object passed to the use everytime a pong is received. When is the last response, Done = true
type Response struct {
	Ping Ping
	Pong Pong
	Done bool
}

//GoPinger is the interface that
type Pinger interface {
	//Ping sends the Ping struct to the mainLoop through the chPing channel
	Send(p Ping)

	//Pong receives responses from the mainLoop
	PongChan() <-chan Response
}

//Implements the context interface
type pinger struct {
	//Channel to send responses to the package User
	Pongch chan Response

	//Channel to receive responses from underlying ICMP Sender
	pongch chan Response

	//WaitGroup
	wg sync.WaitGroup
}
type pingPinger struct {
	pinger Pinger
	ping   Ping
}

func (p pinger) PongChan() <-chan Response {
	return p.Pongch
}

//New Creates a new Pinger.
func New() Pinger {
	return pinger{
		Pongch: make(chan Response),
		pongch: make(chan Response),
	}
}

var (
	chPing   = make(chan pingPinger) //Receives ICMP requests
	chPong   = make(chan Pong)       //Receives ICMP Responses
	chPause  = make(chan struct{})   //Receives Pause commands
	chResume = make(chan struct{})   //Receives Resume commands
	ctrl     = struct {              //Status of mainLoop. Also mutex for status change synchronization
		paused  bool
		running bool
		sync.Mutex
	}{paused: false, running: false}
)

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
func (p pinger) Send(ping Ping) {

	if ping.Count >= 0 && int(ping.Sent) >= ping.Count {

		//Sending the Done Response when number of ping.Sent is equals the number of ping.Count
		p.Pongch <- Response{ping, Pong{}, true}

		//Closing the channel if no ping requests is left on the given context.
		ctxPingCounter[ping.pinger.PongChan()]--
		if ctxPingCounter[ping.pinger.PongChan()] <= 0 { //If counter is <= 0 then we should close the channel on context
			delete(ctxPingCounter, ping.pinger.PongChan()) //Deleting the map key.
			close(ping.pinger.pongChan())                  //Closing the channel on context
		}
	} else {
	}

	chPing <- pingPinger{p, ping}
}

//mainLoop controls the schedule of ICMP requests (pings) and ICMP responses (pongs). And leads with timeouts
func mainLoop() {

	var (
		//totalPings counts the number of pings that have been sent from this engine
		totalPings uint64
	)

	//The main loop starts here
	for {
		select {
		case pp := <-chPing:
			//Sending the Done Response when number of ping.Sent is equals the number of ping.Count
			if pp.ping.Count >= 0 && int(pp.ping.Sent) >= pp.ping.Count {
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
			_ = pong
		/*
			if recvchans[pong.Seq] != nil {
				recvchans[pong.Seq] <- pong
				close(recvchans[pong.Seq])
				recvchans[pong.Seq] = nil
			}
		*/
		case <-chPause:
			<-chResume //Blocks until a resume command is received
		}
	}
}

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
