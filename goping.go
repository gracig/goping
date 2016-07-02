package goping

import (
	"math"
	"sync"
	"time"

	"github.com/gracig/goshared/log"
)

const (
	//IPV4 is the ip version to choose in the Ping.IPVersion field
	IPV4 = iota
	//IPV6 is the ip version to choose in the Ping.IPVersion field
	IPV6
	//ICMP is the IP Protocol to choose in the Ping.IPProto field
	ICMP
	//UDP is the IP Protocol to choose in the Ping.IPProto field
	UDP
	//TCP is the IP Protocol to choose in the Ping.IPProto field
	TCP
	//MAXSEQUENCE is the max number of the ICMP sequence field
	MAXSEQUENCE = 65536
)

//Ping is the ICMP Request to be done
type Ping struct {
	IPVersion uint
	IPProto   uint
	Data      map[string]string
	Host      string
	Timeout   uint
	Interval  uint
	Count     uint

	Sent   float64
	Recv   float64
	Failed float64
	AvgRtt float64
	MaxRtt float64
	MinRtt float64

	ctx Context
}

//Pong is the response for each Ping.
type Pong struct {
	Seq         uint
	Size        uint
	RTT         float64
	ICMPType    uint
	ICMPSubtype uint
	Timedout    bool
}

//Response is the object passed to the use everytime a pong is received. When is the last response, Done = true
type Response struct {
	Ping
	Pong
	Done bool
}

//Context is the interface to be used in this library.
//ResponseChannel as are used inside contexts, this allow the library to be used concurrently
type Context interface {
	//Channel where the Response objects will be sent
	RecvChannel() <-chan Response

	//Internal pingpong channge
	sendChannel() chan<- Response
}

//Implements the context interface
type context struct {
	channel chan Response
}

func (c context) RecvChannel() <-chan Response {
	return c.channel
}
func (c context) sendChannel() chan<- Response {
	return c.channel
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

//NewContext Creates a new Context to be used inside Add().
func NewContext() Context {
	return context{
		channel: make(chan Response),
	}
}

//Add Adds a new ping job to be done. The context can be obtained by the NewContext() function
//The Context contains the Response channel and must be read in a for range as soon as possible to avoid goroutine contention
//for resp := range Context.RecvChannel(){
//	//Consumes the response
//}
func Add(ping Ping, ctx Context) error {
	ping.ctx = ctx
	chPing <- ping
	return nil
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
		log.Info.Println("goping was succesfully paused")
	} else {
		log.Warn.Println("You requested to pause  an already paused goping")
	}
}

//Resume resumes the main loop, if it is paused.
func Resume() {
	ctrl.Lock()
	defer ctrl.Unlock()
	if ctrl.paused {
		chResume <- struct{}{}
		ctrl.paused = false
		log.Info.Println("goping was succesfully resumed")
	} else {
		log.Warn.Println("You requested to resume a non paused goping")
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
				ctxPingCounter[ping.ctx.RecvChannel()]++
			}
			//Sending the Done Response when number of ping.Sent is equals the number of ping.Count
			if uint(ping.Sent) >= ping.Count {
				ping.ctx.sendChannel() <- Response{ping, Pong{}, true}
				//Closing the channel if no ping requests is left on the given context.
				ctxPingCounter[ping.ctx.RecvChannel()]--         //Decrements the context Ping Counter
				if ctxPingCounter[ping.ctx.RecvChannel()] <= 0 { //If counter is <= 0 then we should close the channel on context
					delete(ctxPingCounter, ping.ctx.RecvChannel()) //Deleting the map key.
					close(ping.ctx.sendChannel())                  //Closing the channel on context
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
							Seq:      uint(sequence),
							RTT:      math.NaN(),
							Timedout: true,
						}
					case pong = <-chrecv: //Received Pong
					}
					ping.ctx.sendChannel() <- Response{ping, pong, false} //Send Response to context channel. Done is false
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
