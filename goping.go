package goping

import (
	"math"
	"sync"
	"sync/atomic"
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

//Pong is the rsponse for each Ping.
type Pong struct {
	Seq         uint
	Size        uint
	RTT         float64
	ICMPType    uint
	ICMPSubtype uint
	Timedout    bool
}

//PingPong is the object passed to the use everytime a pong is received. When is the last response, Done = true
type PingPong struct {
	Ping
	Pong
	Done bool
}

//Context is the interface to be used in this library.
//PingPongChannel as are used inside contexts, this allow the library to be used concurrently
type Context interface {
	//Channel where the PingPong objects will be sent
	PingPongChannel() <-chan PingPong
	//Internal function
	pingPongChannel() chan PingPong
	//Internal function
	addPing(delta int32)
	//Internal function
	pingCount() int32
}

//Implements the context interface
type context struct {
	chpp  chan PingPong
	count int32
	sync.Mutex
}

func (c *context) PingPongChannel() <-chan PingPong {
	return c.chpp
}
func (c *context) pingPongChannel() chan PingPong {
	return c.chpp
}
func (c *context) addPing(delta int32) {
	atomic.AddInt32(&c.count, delta)
}
func (c *context) pingCount() int32 {
	return atomic.LoadInt32(&c.count)
}

//NewContext Creates a new Contet to be used in the Add function
func NewContext() Context {
	return &context{
		chpp: make(chan PingPong),
	}
}

//Used to control the main loop execution at the Run function
type control struct {
	paused bool
	sync.Mutex
}

var (
	//Channels to control
	chPing   = make(chan Ping)
	chPong   = make(chan Pong)
	chPause  = make(chan struct{})
	chResume = make(chan struct{})
	chDone   = make(chan struct{})
	ctrl     = control{paused: false}
	seq      uint64
)

//Add Adds a new ping job to be done. The context  should be passed returns errors if ping  is invalid
func Add(ping Ping, ctx Context) error {
	ping.ctx = ctx
	chPing <- ping
	return nil
}

//Run executes the main loop. coordinates all the pings and pongs, and timeouts, and pause and resume
func Run() {
	//Work with all states inside this function

	var recvchans = make([]chan Pong, MAXSEQUENCE, MAXSEQUENCE)

	for {
		select {

		case ping := <-chPing:
			log.Debug.Println("Called Ping")

			//Verifies if the number of sent pings is less then the number of pings we need to send
			if uint(ping.Sent) < ping.Count {

				//If it is the first ping, adding context count
				if ping.Sent == 0 {
					ping.ctx.addPing(1)
				}

				//Prepare Ping
				sequence := atomic.AddUint64(&seq, 1) % MAXSEQUENCE
				if recvchans[sequence] != nil {
					close(recvchans[sequence])
					recvchans[sequence] = nil
				}
				recvchans[sequence] = make(chan Pong, 1) //The number avoid the channel to hold state

				//Make a ping
				ping.Sent++

				//Wait a response
				go func(sequence uint64, ping Ping, chrecv chan Pong) {

					//Creates the timer to implement the ping.Interval field
					wait := time.NewTimer(time.Millisecond * time.Duration(ping.Interval))

					//Wait for a Pong or times out
					var pong Pong
					tout := time.NewTimer(time.Millisecond * time.Duration(ping.Timeout))
					select {
					case <-tout.C:
						pong = Pong{
							Seq:      uint(sequence),
							RTT:      math.NaN(),
							Timedout: true,
						}
						if log.DebugEnabled {
							log.Debug.Printf("Timed out: %v %v", ping, pong)
						}
					case pong = <-chrecv:
					}

					//Send Ping Pong to user in a goroutine to not block function
					ping.ctx.pingPongChannel() <- PingPong{ping, pong, false}

					//Waits interval time
					<-wait.C

					//Send ping back to ping channel
					chPing <- ping

				}(sequence, ping, recvchans[sequence])

			} else {

				//Sends the last PingPong, with a zero value Pong and the field Done as true
				ping.ctx.pingPongChannel() <- PingPong{ping, Pong{}, true}

				//Decrement the ping counter
				ping.ctx.addPing(-1)

				//Verifies if ping counter is equals zero. this indicates that the pingpongchannel should be closed
				if ping.ctx.pingCount() == 0 {
					close(ping.ctx.pingPongChannel())
				}

				//Verifies if ping counter is less than zero, this is an abnormal situation
				if ping.ctx.pingCount() < 0 {
					log.Severe.Fatalln("pingCount cannot be, never, less than zero. something is very wrong")
				}

			}

		case <-chPong:
			if log.DebugEnabled {
				log.Debug.Printf("Called pong\n")
			}
		case <-chPause:
			if log.DebugEnabled {
				log.Debug.Printf("Called Pause\n")
			}

			<-chResume
			if log.DebugEnabled {
				log.Debug.Printf("Called Resume\n")
			}
		case <-chDone:
			//Finalize and exit
		}
	}
}

//Pause pauses the main execution
func Pause() {
	ctrl.Lock()
	defer ctrl.Unlock()
	if !ctrl.paused {
		chPause <- struct{}{}
		ctrl.paused = true
		log.Debug.Println("goping was succesfully paused")
	} else {
		log.Debug.Println("You requested to pause  an already paused goping")
	}
}

//Resume resumes the pause command
func Resume() {
	ctrl.Lock()
	defer ctrl.Unlock()
	if ctrl.paused {
		chResume <- struct{}{}
		ctrl.paused = false
		log.Debug.Println("go ping was succesfully resumed")
	} else {
		log.Debug.Println("You requested to resume a non paused goping")
	}
}
