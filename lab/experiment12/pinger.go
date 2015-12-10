package ggping

import (
	"container/list"
	"container/ring"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	g_IDENTIFIER   int     = os.Getpid() & 0xffff  //Used to set the ID field of the ICMP Message
	g_SEQUENCE     counter = counter{max: 0x10000} //Used to set the next sequence field of the ICMP Message
	g_REQUESTID    counter                         //Used to set the next Ping ID
	g_PKT_SENT     counter
	g_PKT_RECEIVED counter

	g_PINGER_RING *ring.Ring      = ring.New(int(g_SEQUENCE.max))    //The  pinger ring
	g_REPLY_ARRAY []*pinger       = make([]*pinger, g_SEQUENCE.max)  //The array that receives replies indexed by the sequence
	g_REPLY_MAP   map[int]*pinger = make(map[int]*pinger)            //The array that receives replies indexed by the sequence
	g_POOL_SIZE   int             = 64                               //Limit the ammount of "simultaneous" ping
	g_POOL_AVAIL  chan struct{}   = make(chan struct{}, g_POOL_SIZE) //Limit the ping pool size to g_SEQUENCE.max
	g_PINGER_POOL *sync.Pool      = &sync.Pool{                      //Initializes the pinger Pool
		New: func() interface{} {
			return &pinger{
				chWhen: make(chan time.Time, 1),
				chPong: make(chan *pinger, 1),
			}
		},
	}
	g_ICMP_PINGER         icmppinger //The Singleton icmppinger
	g_CHECK_RESPONSE_CHAN = make(chan int)
	g_CHECK_RESPONSE_ONCE sync.Once
)

func checkResponse() {
	go func() {
		ps := pstatus{r: g_PINGER_RING}
		var p *pinger
		var t time.Time
		for seq := range g_CHECK_RESPONSE_CHAN {
			findPinger(&ps, seq)
			if ps.r.Value == nil {
				log.Fatal("Sequence Address in Pinger Ring has a nil value")
			}
			p = ps.r.Value.(*pinger)
			select {
			case p = <-p.chPong:
				p.Timedout = false
			case t = <-time.After(p.Settings.Timeout - time.Now().Sub(p.WhenSent)):
				select {
				case p = <-p.chPong:
					p.Timedout = false
				default:
					p.Timedout = true
					p.WhenRecv = t
				}
			}
			p.chReply <- p
		}
	}()
}

func SetPoolSize(sz int) {
	g_POOL_SIZE = sz
	g_POOL_AVAIL = make(chan struct{}, sz)
}
func GetPoolSize() int {
	return g_POOL_SIZE
}

type pstatus struct {
	pos int
	r   *ring.Ring
}

func findPinger(p *pstatus, pos int) {
	var clockwise bool = true
	d := pos - p.pos
	if d < 0 {
		d *= -1
		clockwise = false
	}
	d2 := int(g_SEQUENCE.max) - d
	if d2 < d {
		clockwise = !clockwise
		d = d2
	}
	if !clockwise {
		d *= -1
	}
	p.r = p.r.Move(d)
	p.pos = pos
}

//A Ping Request
type request struct {
	Target   *Target   //The target host
	Settings *Settings //The Ping Settings
	Seq      int       //The ping sequence to be used
}

//A Ping Reply
type reply struct {
	WhenSent time.Time //The time the ping was sent
	WhenRecv time.Time //The time the ping was Received
	Timedout bool      //Indicates if this response was timed out
	Peer     net.IP    //The IP that answered the ping
	ICMPType byte
	ICMPCode byte
}

func (r *reply) error() error {
	if r.Timedout {
		return fmt.Errorf("Request timeout")
	}
	switch r.ICMPType {
	case 0:
		return nil
	default:
		return fmt.Errorf("Error received Type:[%v] Code:[%v] from [%v]", r.ICMPType, r.ICMPCode, r.Peer)
	}
	return nil
}

func (r *reply) rtt() float64 {
	return float64(r.WhenRecv.Sub(r.WhenSent)) / float64(time.Millisecond)
}

//The Ping Object that will be used in the sync.Pool mechanism
type pinger struct {
	request
	reply
	chReply     chan *pinger
	chPong      chan *pinger
	chWhen      chan time.Time
	cycle       int
	packetbytes []byte
	e           *list.Element
	r           *ring.Ring
}

//The ping function
func (p *pinger) ping() {
	g_CHECK_RESPONSE_ONCE.Do(checkResponse)

	//Retrieves the singleton ICMP Pinger
	//TODO: To Implement ping for UDP and for other OSes.
	pinger := getICMPPinger()
	p.WhenSent = pinger.ping(p)
	//p.WhenSent = time.Now()

}

//A Thread safe counter
type counter struct {
	uint64
	max uint64
	sync.Mutex
}

//The counter iterator
func (seq *counter) next() uint64 {
	seq.Lock()
	defer seq.Unlock()
	if seq.max > 0 {
		seq.uint64 = (seq.uint64 + 1) % seq.max
	} else {
		seq.uint64 = (seq.uint64 + 1)
	}
	return seq.uint64
}

//The OS Pinger Interface. Should be implemented by all OSes. Right now only linux is supported
type ospinger interface {
	ping(*pinger) time.Time
	init()
}

//Represents an icmp Pinger. Later we would have an UDP Pinger
type icmppinger struct {
	ospinger
	sync.Once
}

//Get pinger from the pool
func getPingerFromPool() *pinger {
	p := g_PINGER_POOL.Get().(*pinger)
	return p
}

//Return pinger to the pool
func putPingerOnPool(p *pinger) {
	//Reset some variables
	p.chReply = nil
	p.packetbytes = nil
	p.reply.Timedout = false
	p.Target = nil
	p.e = nil
	p.r = nil
	p.Settings = nil
	p.packetbytes = nil
	select {
	case <-p.chPong:
	default:
	}
	select {
	case <-p.chWhen:
	default:
	}
	g_PINGER_POOL.Put(p)

}

//Returns a singleton icmp pinger
func getICMPPinger() *icmppinger {
	g_ICMP_PINGER.Do(
		func() {
			switch os := runtime.GOOS; os {
			case "linux":
				g_ICMP_PINGER.ospinger = new(icmpLinux)
			default:
				log.Fatal("No pinger associated with GOOS: %s", os)
			}
			g_ICMP_PINGER.init()
		},
	)
	return &g_ICMP_PINGER
}
