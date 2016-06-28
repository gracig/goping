package goping

import (
	"sync"
	"sync/atomic"
	"time"
	"math"

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
	Host  string
	Count uint
	Sent  uint
	Recv  uint
}

type Pong struct {
	seq uint64
	RTT float64
}

type seqPing struct {
	ping     Ping
	onPong   func(ping Ping, pong Pong)
	onFinish func(ping Ping)
}

var (
	chPing   chan seqPing  = make(chan seqPing)
	chPong   chan Pong     = make(chan Pong)
	chTicker *time.Ticker  = time.NewTicker(time.Second * 1)
	chPause  chan struct{} = make(chan struct{})
	chResume chan struct{} = make(chan struct{})
	chDone   chan struct{} = make(chan struct{})

	paused  bool = false
	pausedM sync.Mutex
	seq     uint64
)

func Add(ping Ping, onPong func(ping Ping, pong Pong), onFinish func(ping Ping)) error {
	chPing <- seqPing{ping, onPong, onFinish}
	return nil
}

func Run() {
	//Work with all states inside this function

	var recvchans []chan Pong = make([]chan Pong, MAX_SEQUENCE, MAX_SEQUENCE)

	for {
		select {
		case p := <-chPing:
			log.Info.Printf("Called ping\n")
			if p.ping.Sent < p.ping.Recv {
				//Prepare Ping
				sequence := atomic.AddUint64(&seq, 1) % MAX_SEQUENCE
				p.ping.Sent++
				if (recvchans[sequence] != nil){
					close (recvchans[sequence])
					recvchans[sequence] = nil
				}
				recvchans[sequence] = make (chan Pong,1) //The number avoid the channel to hold state

				//Make a ping

				//Wait a response
				go func(sequence uint64 ,p seqPing, chrecv chan Pong){
					tout :=  time.NewTimer(time.Second * 3)
					var pong Pong = Pong{ sequence, math.NaN()}
					select {
					case <-tout.C:
						log.Warn.Println("Timed out")
					case pong= <-chrecv:
					}
					p.onPong(p.ping, pong)
				}(sequence, p , recvchans[sequence])
			} else {
				//Finish Ping

			}
		case <-chPong:
			log.Info.Printf("Called pong\n")
		case <-chTicker.C:
			log.Info.Printf("Called ticker\n")
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
	pausedM.Lock()
	defer pausedM.Unlock()
	if !paused {
		chPause <- struct{}{}
		paused = true
		log.Info.Println("goping was succesfully paused")
	} else {
		log.Warn.Println("You requested to pause  an already paused goping")
	}
}

func Resume() {
	pausedM.Lock()
	defer pausedM.Unlock()
	if paused {
		chResume <- struct{}{}
		paused = false
		log.Info.Println("go ping was succesfully resumed")
	} else {
		log.Warn.Println("You requested to resume a non paused goping")
	}
}
