package ggping

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"testing"
	"time"
)

type pingTaskRow struct {
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
	{Iter: 1, To: "localhost", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
	{Iter: 1, To: "www.google.com", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
	{Iter: 1, To: "192.168.0.1", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
	{Iter: 1, To: "www.terra.com.br", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
	{Iter: 1, To: "www.uol.com.br", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
	{Iter: 1, To: "www.ig.com.br", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
}

func TestCoordinator(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	pings := 3000
	pinger := newLinuxPinger(100 * time.Microsecond)

	send := make(chan *EchoRequest, pings)
	recv := make(chan *EchoRequest)
	appendpong := make(chan *Pong)
	done := make(chan *Summary)

	go pinger.Start(send, recv)

	var wg sync.WaitGroup
	start := make(chan struct{})
	go func() {
		for _, p := range pingTaskTable {
			for i := 0; i < p.Iter; i++ {
				if pingTask, err := NewPingTask(p.To, p.Timeout, p.Interval, p.TOS, p.TTL, p.REQUESTS, p.PKTSZ, p.UserMap); err != nil {
					log.Println("Error creating pingTask:", err)
				} else {
					debug.Println("Send first request for ", pingTask.to, " ", pingTask.id)
					if echo, err := pingTask.makeEchoRequest(false); err != nil {
						info.Println("Could not make Echo Request")
					} else {
						wg.Add(1)
						send <- echo
						if start != nil {
							start <- struct{}{}
							start = nil
						}
					}
				}
			}
		}
	}()

	go func() {
		defer close(appendpong)
		for echo := range recv {
			//Send next request if any
			go func(echo *EchoRequest) {
				if !echo.next {
					if echo.index+1 < echo.pingTask.requests {
						echo.next = true
						time.Sleep(echo.pingTask.interval - time.Now().Sub(echo.when))
						debug.Println("Send a new request for ", echo.pingTask.to)
						if echo, err := echo.pingTask.makeEchoRequest(false); err != nil {
							log.Println("Could not create Echo Request ", echo.pingTask.id)
						} else {
							send <- echo
						}
					}
				}

				//Retrieve the response object
				var pong *Pong

				select {
				case pong = <-echo.pongchan:
				case <-time.After(echo.pingTask.timeout - time.Now().Sub(echo.when)):
					pong = &Pong{
						EchoStart: echo.when,
						EchoEnd:   time.Now(),
						Err:       fmt.Errorf("Response Timed Out"),
					}
				}

				//Append the response object if received or timed out
				pong.idx = echo.index
				pong.pi = echo.pingTask
				appendpong <- pong
			}(echo)
		}
	}()

	go func() {
		defer close(done)
		sm := make(map[*PingTask]*Summary)
		for pong := range appendpong {
			debug.Println(pong.EchoStart, pong)

			if pong.idx == 0 {
				sm[pong.pi] = &Summary{
					PingTask:   pong.pi,
					Min:        -1,
					Max:        -1,
					Avg:        -1,
					PacketLoss: 0,
					//		Pongs:      make([]*Pong, pong.pi.requests),
				}
			}
			s := sm[pong.pi]
			s.Sent++
			//s.Pongs[pong.idx] = pong
			if pong.Err != nil {
				s.Failed++
			} else {
				s.Success++
				rtt := float64(pong.EchoEnd.Sub(pong.EchoStart)) / float64(time.Millisecond)
				if s.Min == -1 || s.Min > rtt {
					s.Min = rtt
				}
				if s.Max == -1 || s.Max < rtt {
					s.Max = rtt
				}
				s.RttSum += rtt
				s.Avg = s.RttSum / s.Success
			}
			s.PacketLoss = (s.Failed / s.Sent) * 100.0

			if pong.idx+1 == pong.pi.requests {
				done <- s
			}
		}

	}()

	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for s := range done {
			info.Println(s.Min, s.Max, s.Avg, s.Sent, s.Success, s.Failed, s.PacketLoss, s.PingTask.to, s.PingTask.toaddr, s.PingTask.id)
			//info.Printf("%v-Finish pings for %v ready to summarize ", p.id, p.to)
			wg.Done()
		}
	}()
	<-start
	wg.Wait()
	close(send)
	<-finished
}
