package ggping

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var c int32

func RunCh(targets []*Target, s Settings) (chreply chan *EchoReply) {
	chreply = make(chan *EchoReply)

	go func() {
		defer close(chreply)
		RunFn(targets, s, func(r *EchoReply) { chreply <- r })
		//RunFn(targets, s, nil)
	}()

	return
}

func RunFn(targets []*Target, s Settings, fnEchoReply func(*EchoReply)) {

	chrep := make(chan *pinger, len(targets)*s.CyclesCount)
	chreq := make(chan *pinger)
	chdone := make(chan struct{})
	packetbytes := make([]byte, s.Bytes)

	wait := s.CyclesInterval / time.Duration(len(targets)*2)
	var wg sync.WaitGroup
	wg.Add(len(targets) * s.CyclesCount)

	go func(chrep, chreq chan *pinger, chdone chan struct{}, wg *sync.WaitGroup) {

		//Iterate over the CyclesCount Settings
		for cycle := 0; cycle < s.CyclesCount; cycle++ {
			info.Printf("Starting Cycle:%v\n", cycle)

			//Tag the time the cycle started
			cycleStart := time.Now()

			//Iterates over all targets
			for _, t := range targets {

				g_POOL_AVAIL <- struct{}{}

				//Retrieves a Pinger from the pool. Blocks if no pinger is available.
				p := getPingerFromPool()

				//Creates a request object to be sent to the pinger
				p.cycle = cycle + 1
				p.Settings = &s
				p.Target = t
				p.chReply = chrep
				p.packetbytes = packetbytes

				chreq <- p
				time.Sleep(wait) //Avoid overload the kernel with ping requests

			}

			//Waits for the cycle interval before start other cycle
			time.Sleep(s.CyclesInterval - time.Now().Sub(cycleStart))
			//time.Sleep(s.CyclesInterval)
		}
		wg.Wait()
		chdone <- struct{}{}
	}(chrep, chreq, chdone, &wg)

	tk := time.NewTicker(time.Second)
	ps := pstatus{r: g_PINGER_RING}

	for {
		select {
		case p := <-chrep:
			<-g_POOL_AVAIL
			if !p.Timedout {
				c = atomic.AddInt32(&c, -1)
			}

			//if fnEchoReply is not null, send the reply to the function
			if fnEchoReply != nil {
				fnEchoReply(
					&EchoReply{
						Bytes:  p.request.Settings.Bytes,
						Err:    p.reply.error(),
						Peer:   p.reply.Peer,
						Rtt:    p.reply.rtt(),
						Seq:    p.request.Seq,
						TTL:    p.request.Settings.TTL,
						Target: p.Target,
						Cycle:  p.cycle,
					},
				)
			}

			p.chReply = nil
			p.r.Value = nil
			putPingerOnPool(p)
			wg.Done()

		case p := <-chreq:
			c = atomic.AddInt32(&c, 1)
			p.Seq = int(g_SEQUENCE.next())
			findPinger(&ps, p.Seq)
			p.r = ps.r
			p.r.Value = p
			p.ping()
			go func(seq int) {
				g_CHECK_RESPONSE_CHAN <- seq
			}(p.Seq)

		case <-tk.C:

		//Receive the done signal
		case <-chdone:
			info.Println("Counter Reference", c)
			return
		}
	}

}

//The target host to be pinged
type Target struct {
	From     *net.IPAddr       //The source ip address
	IP       *net.IPAddr       //The target IP Address
	UserData map[string]string //UserData passed by the user
}

//Ping Settings
type Settings struct {
	Bytes          int           //Byte ammout to send in the icmp Message
	TOS            int           //The DSCP code to set in the IP Header
	TTL            int           //THe time to live value to set in the IP Header
	Timeout        time.Duration //THe maximum ammount of time to wait for an answer
	CyclesCount    int           //How many times to ping the  TargetList
	CyclesInterval time.Duration //The mimnum interval between cycles
}

//The summary of all requests for a target
type Summary struct {
	Target *Target //The target device
	Sent   float64 //The number of ping sents
	Recv   float64 //The number of succesful replies received
	Max    float64 //The maximum rtt
	Min    float64 //The minimum rtt
	Avg    float64 //The average rtt
}

//64 bytes from 200.221.2.45: icmp_seq=2 ttl=53 time=25.291 ms
type EchoReply struct {
	Cycle  int
	Target *Target
	Rtt    float64
	Bytes  int
	Peer   net.IP
	Seq    int
	TTL    int
	Err    error
}

func (r EchoReply) String() string {
	if r.Err != nil {
		return fmt.Sprintf("cycle=%v: %v for icmp_seq=%v", r.Cycle, r.Err, r.Seq)
	} else {
		return fmt.Sprintf("cycle=%v: %v bytes from %v: icmp_seq=%v ttl=%v time=%.3f ms", r.Cycle, r.Bytes, r.Peer, r.Seq, r.TTL, r.Rtt)
	}

}
