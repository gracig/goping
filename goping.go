package goping

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gracig/goshared/log"
)

const (
	IPV4 = iota
	IPV6
	ICMP
	UDP
	TCP
	MAX_SEQUENCE = 65536
)

type Ping struct {
	Data     map[string]string
	Host     string
	Timeout  uint
	Interval uint
	Count    uint

	Sent   float64
	Recv   float64
	Failed float64
	AvgRtt float64
	MaxRtt float64
	MinRtt float64

	ctx Context
}

type Pong struct {
	Seq         uint
	Size        uint
	RTT         float64
	ICMPType    uint
	ICMPSubtype uint
	Timedout    bool
}

//Represent a response Tuple
type PingPong struct {
	Ping
	Pong
	Done bool
}

type Context interface {
	PingPongChannel() <-chan PingPong
	pingPongChannel() chan PingPong
	addPing(delta int)
	pingCount() int
}

type context struct {
	chpp  chan PingPong
	count int
}

func (c *context) PingPongChannel() <-chan PingPong {
	return c.chpp
}
func (c *context) pingPongChannel() chan PingPong {
	return c.chpp
}
func (c *context) addPing(delta int) {
	c.count += delta
}
func (c *context) pingCount() int {
	return c.count
}

func NewContext() Context {
	return &context{
		chpp: make(chan PingPong),
	}
}

type control struct {
	paused bool
	sync.Mutex
}

var (
	//Channels to control
	chPing   chan Ping     = make(chan Ping)
	chPong   chan Pong     = make(chan Pong)
	chPause  chan struct{} = make(chan struct{})
	chResume chan struct{} = make(chan struct{})
	chDone   chan struct{} = make(chan struct{})

	ctrl = control{paused: false}

	seq uint64
)

func Add(ping Ping, ctx Context) error {
	ping.ctx = ctx
	chPing <- ping
	return nil
}

func Run() {
	//Work with all states inside this function

	var recvchans []chan Pong = make([]chan Pong, MAX_SEQUENCE, MAX_SEQUENCE)

	for {
		select {

		case ping := <-chPing:

			if uint(ping.Sent) < ping.Count {
				//If it is the first ping, adding context count
				if ping.Sent == 0 {
					ping.ctx.addPing(1)
				}

				//Prepare Ping
				sequence := atomic.AddUint64(&seq, 1) % MAX_SEQUENCE
				if recvchans[sequence] != nil {
					close(recvchans[sequence])
					recvchans[sequence] = nil
				}
				recvchans[sequence] = make(chan Pong, 1) //The number avoid the channel to hold state

				//Make a ping
				ping.Sent++

				//Wait a response
				go func(sequence uint64, ping Ping, chrecv chan Pong) {

					//Creates the timer for another ping
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
						log.Warn.Println("Timed out")
					case pong = <-chrecv:
					}

					//Send Ping Pong to user in a goroutine to not block function
					ping.ctx.pingPongChannel() <- PingPong{ping, pong, false}

					//Sends Ping to chPing after wait for interval
					<-wait.C
					chPing <- ping

				}(sequence, ping, recvchans[sequence])

			} else {
				//Finish Ping
				ping.ctx.pingPongChannel() <- PingPong{ping, Pong{}, true}
				ping.ctx.addPing(-1)
				if ping.ctx.pingCount() == 0 {
					close(ping.ctx.pingPongChannel())
				}
			}
		case <-chPong:
			log.Info.Printf("Called pong\n")
		case <-chPause:
			log.Info.Printf("Called Pause\n")
			<-chResume
			log.Info.Printf("Called Resume\n")
		case <-chDone:
			//Finalize and exit
		}
	}
}

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

func Resume() {
	ctrl.Lock()
	defer ctrl.Unlock()
	if ctrl.paused {
		chResume <- struct{}{}
		ctrl.paused = false
		log.Info.Println("go ping was succesfully resumed")
	} else {
		log.Warn.Println("You requested to resume a non paused goping")
	}
}
