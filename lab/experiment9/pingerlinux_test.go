package ggping

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestCoordinator(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	pings := 1000

	pinger := newLinuxPinger(0 * time.Millisecond)
	send := make(chan *Ping)
	recv := make(chan *Ping)

	go pinger.Start(send, recv)

	go func() {
		for i := 0; i < pings; i++ {
			if ping, err := NewPing("localhost", 2, 1, 0, 64, 1, 100, nil); err != nil {
				fmt.Println("Error creating ping:", err)
			} else {
				debug.Println("Sending ping", ping)
				send <- ping
			}
		}
	}()

	for i := 0; i < pings; i++ {
		ping := <-recv
		debug.Println("Received Ping through recv channel", ping)
		select {
		case pong := <-ping.pongchan:
			debug.Println("Received pong on pongchan:", ping.pongchan)
			close(ping.pongchan)
			if rtt, err := pong.Rtt(); err != nil {
				info.Println("Error receiving Ping\n")
			} else {
				//		if rtt > 1 || rtt < 0 {
				info.Println(ping.when, ping.to, pong.Sequence, rtt)
				//		}
			}

		default:
			debug.Println("NotReady pongchan:", ping.pongchan)
			t := time.Now()
			if t.Sub(ping.when) > (time.Duration(ping.timeout) * time.Second) {
				info.Println(ping.when, t, " Request Timeout")
			} else {
				//Put Reply back on channel, because it has not timed out yet
				i--
				go func(ping *Ping) {
					recv <- ping
				}(ping)
			}
		}
	}
	close(recv)
}
