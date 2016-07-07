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
	/*Config */
	Config

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
	pongch chan Response
	ping   Ping
}

func (p *pinger) PongChan() <-chan Response {
	return p.Pongch
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

func (p *pinger) Wait() {
	p.wg.Wait()
}

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
func (p *pinger) Send(ping Ping) {

	//Start goroutine that waits for a ping response or timeouts
	go func(p *pinger, ping Ping) {
		var r Response
		wait := time.NewTimer(time.Millisecond * time.Duration(ping.Interval))
		tout := time.NewTimer(time.Millisecond * time.Duration(ping.Timeout))
		select {
		case r = <-p.pongch:
		case <-tout.C:
			r.Ping = ping
			r.Pong = Pong{
				RTT: math.NaN(),
				Err: fmt.Errorf("Timeout"),
			}
		}

		//Send response to user in a go routine to avoid blocking
		go func() {
			p.Pongch <- r
			p.wg.Done()
		}()

		//If not Done, send another ping request
		if !r.Done {
			//Waits for the ping interval
			<-wait.C
			//Calls p.Send
			p.Send(ping)
		}
	}(p, ping)

	p.wg.Add(1)
	if ping.Count >= 0 && int(ping.Sent) >= ping.Count {
		//Sending the Done Response when number of ping.Sent is equals the number of ping.Count
		p.pongch <- Response{ping, Pong{}, true}
	} else {
		//Sanitize Ping

		//If ping.Count <0 or Sent is Less than Count, send ping
		ping.Sent++
		chPing <- pingPinger{p.pongch, ping}
	}
}

//mainLoop controls the schedule of ICMP requests (pings) and ICMP responses (pongs). And leads with timeouts
func mainLoop(isf ICMPSenderFactory) {

	var (
		//totalPings counts the number of pings that have been sent from this engine
		totalPings uint64
	)

	//The main loop starts here
	for {
		select {
		case pp := <-chPing:
			//Incrementing ping counters
			totalPings++

			//Generating a sequence number based on the local counter totalPings
			sequence := totalPings % MAXSEQUENCE
			pp.ping.ICMPMessage.Body.(*icmp.Echo).Seq = int(sequence)

			isf.GetICMPSender(pp.ping.IPVersion, pp.ping.IPProto)

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
func Run(isf ICMPSenderFactory) {
	ctrl.Lock()
	defer ctrl.Unlock()
	if !ctrl.running {
		ctrl.running = true
		go mainLoop(isf)
	}
}
