package goping

import (
	"log"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"
)

type pingTaskRow struct {
	Peak     int
	Iter     int
	To       string
	Timeout  time.Duration
	Interval time.Duration
	TOS      int
	TTL      int
	REQUESTS int
	PKTSZ    uint
	UserMap  map[string]string
}

var pingTaskTable = []pingTaskRow{
	/*
		{Peak: 10, Iter: 2000, To: "localhost", Timeout: 3 * time.Second, Interval: 0000 * time.Nanosecond, TOS: 0, TTL: 64, REQUESTS: 100, PKTSZ: 10, UserMap: nil},
		{Peak: 0, Iter: 1, To: "www.google.com", Timeout: 3 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
		{Peak: 10, Iter: 1, To: "192.168.0.1", Timeout: 3 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
		{Peak: 10, Iter: 1, To: "www.terra.com.br", Timeout: 3 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
		{Peak: 10, Iter: 1, To: "www.uol.com.br", Timeout: 3 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
		{Peak: 10, Iter: 1, To: "www.ig.com.br", Timeout: 3 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
	*/
	{Peak: 10, Iter: 1, To: "41.0.0.10", Timeout: 3 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
}

var wg sync.WaitGroup

func TestCoordinator(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	ticker := time.NewTicker(time.Millisecond * 1000)
	go func() {
		var dtSent0, dtRecv0 uint64 = g_PKT_SENT.uint64, g_PKT_RECEIVED.uint64
		for range ticker.C {
			var dtSent1, dtRecv1 uint64 = g_PKT_SENT.uint64, g_PKT_RECEIVED.uint64
			info.Printf("Sent: %v pkts/s Recv: %v pkts/s\n", (dtSent1 - dtSent0), (dtRecv1 - dtRecv0))
			dtSent0, dtRecv0 = dtSent1, dtRecv1
		}
	}()

	//Controls the interval betwen eavh pibg sent over the socket
	pinger := newLinuxPinger(00 * time.Nanosecond)

	send := make(chan EchoRequest)

	//Starts the pinger
	go pinger.Start(send)

	//GoRoutine to create and send all the pingtasks
	for _, p := range pingTaskTable {
		for i := 0; i < p.Iter; i++ {
			//Create the task object and wait for response
			if task, err := NewPing(p.To); err != nil {
				log.Println("Error creating pingTask:", err)
				continue
			} else {
				task.Times = p.REQUESTS
				task.Interval = p.Interval
				task.Timeout = p.Timeout
				task.PeakTh = p.Peak
				wg.Add(1)
				go pingOverChannel(task, send)
			}
		}
	}
	wg.Wait()

}
func pingOverChannel(task *Ping, send chan EchoRequest) {
	defer wg.Done()

	//var requests [100]EchoRequest
	//requests := make([]EchoRequest, task.Times)
	//replies := make([]EchoReply, task.Times)

	pongchannel := make(chan *EchoReply, task.Times)

	//Send the EchoRequests using the task.Interval Duration
	var we sync.WaitGroup
	we.Add(task.Times)
	//Closes the pong channel
	go func() {
		we.Wait()
		close(pongchannel)
	}()

	//Gets the response
	go func() {
		for i := 0; i < task.Times; i++ {

			var err error
			var echo EchoRequest
			if echo, err = task.makeEchoRequest(); err != nil {
				log.Println("Erro creating Echo Request")
			}
			send <- echo
			whensent := <-echo.chwhen
			go func(i int, echo *EchoRequest, whensent time.Time) {
				defer we.Done()
				var reply EchoReply
				select {
				case reply := <-echo.chreply:
				case reply := <-echo.chtimeout:
				}
				reply.WhenSent = whensent
				pongchannel <- &reply
			}(i, &echo, whensent)

			if i+1 < task.Times {
				time.Sleep(task.Interval - time.Now().Sub(whensent)) //Sleeping for the next ping
			}
		}
	}()

	//peaks := make([]float64, 5)
	var peaks []float64
	var min, sum, max, sumpctl, countpctl, pctl, succeded, failed, sent float64

	var peakSize int
	if task.PeakTh > 0 {
		peakSize = (task.Times / task.PeakTh)
	}
	peakSize++

	for pp := range pongchannel {
		sent++
		if pp.Type != 0 {
			debug.Printf("ERROR: Ping Error. Type:%v Code:%v To:%v\n", pp.Type, pp.Code, task.To)
			failed++
		} else {
			succeded++
			countpctl++
			rtt := float64(pp.WhenRecv.Sub(pp.WhenSent)) / float64(time.Millisecond)
			debug.Println("OK:", task.To, rtt, task.Id)
			if min == 0 || min > rtt {
				min = rtt
			}
			if max == 0 || max < rtt {
				max = rtt
			}
			sum += rtt
			sumpctl += rtt

			peaks = append(peaks, rtt)
			sort.Float64s(peaks)
			if len(peaks) > peakSize {
				peaks = peaks[1:peakSize]
			}
		}
	}
	pctl = max
	if len(peaks) > 0 {
		for i, rtt := range peaks {
			if i > 0 {
				countpctl--
				sumpctl -= rtt
			}
		}
		pctl = peaks[0]
	}

	info.Printf("Sent:%v Ok:%v Fails:%v Min:%.3f Max:%.3f Avg:%.3f for: %v (%v) \n", sent, succeded, failed, min, max, sum/succeded, task.To, task.Id)
	debug.Printf("PeakTh: %v Percentil: %.3f AvgPctl:%.3f  \n", task.PeakTh, pctl, sumpctl/countpctl)

}
