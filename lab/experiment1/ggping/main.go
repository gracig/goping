package main

import (
	"time"

	"github.com/gersongraciani/goping"
)

const repeatGroup int = 100
const maxWorkers int = 100

//2015/10/15 00:15:05 Ping Unmatched Response: PING farts.com (184.168.164.84) 100(128) bytes of data.
//2015/10/15 00:15:05 Ping Unmatched Response: PING e10088.dspb.akamaiedge.net (23.213.202.147) 100(128) bytes of data.
//2015/10/15 00:15:05 Ping Unmatched Response: PING cs60.can.transactcdn.com (192.16.31.62) 100(128) bytes of data.
//2015/10/15 00:15:06 Ping Unmatched Response: PING www.gtm.nytimes.com (170.149.161.130) 100(128) bytes of data.
//2015/10/15 00:15:06 Ping Unmatched Response: PING e2094.b.akamaiedge.net (23.11.248.19) 100(128) bytes of data.
//2015/10/15 00:15:06 Ping Unmatched Response: PING www.google.com (173.194.205.103) 100(128) bytes of data.

func main() {
	requests := []*goping.PingRequest{
		//		{To: "www.cnn.com", Tos: 16, Timeout: 1, MaxPings: 10, MinWait: 0.1, Percentil: 90, UserMap: map[string]string{"company": "cnn"}},
		//		{To: "www.microsoft.com", Timeout: 1, MaxPings: 10, MinWait: 1, Percentil: 90, UserMap: map[string]string{"company": "microsoft"}},
		//		{To: "www.dell.com", Timeout: 1, MaxPings: 10, MinWait: 1, Percentil: 90, UserMap: map[string]string{"company": "dell"}},
		//		{To: "www.nytimes.com", Timeout: 1, MaxPings: 10, MinWait: 1, Percentil: 90, UserMap: map[string]string{"company": "nytimes"}},
		//		{To: "www.avaya.com", Timeout: 1, MaxPings: 10, MinWait: 1, Percentil: 90, UserMap: map[string]string{"company": "avaya"}},
		//		{To: "www.google.com", Timeout: 1, MaxPings: 10, MinWait: 0.5, Percentil: 90, UserMap: map[string]string{"company": "google"}},
		{To: "localhost", Timeout: 1, MaxPings: 10, MinWait: 1, Percentil: 90, UserMap: map[string]string{"company": "local"}},
		//		{To: "UnknownHost", Timeout: 1, MaxPings: 10, MinWait: 0, Percentil: 90, UserMap: map[string]string{"company": "nocompany"}},
	}

	startTime := time.Now()
	//pongs := make([]*goping.Pong, 0, repeatGroup*len(requests))
	pongs := make(chan *goping.Pong)
	go func() {
		defer close(pongs)
		for i := 0; i < repeatGroup; i++ {
			for _, request := range requests {
				pongs <- goping.Ping(*&request)
			}
		}
	}()
	var pongsCount int
	for pong := range pongs {
		pongsCount++
		debug.Printf("Startint reading pongs %v", pong.Request)
		<-pong.Done
		if pong.Error != nil {
			debug.Printf("Request Error: %v %v", pong.Request, pong.Error)
			continue
		} else {
			debug.Printf("Request OK: %v %v", pong.Request, pong.Summary)
		}
		for _, v := range pong.Responses {
			debug.Printf("\tResponse: [%v]", v)
		}
	}

	debug.Printf("End of Program, elapsed time: %v workers:%v requests:%v", time.Since(startTime), maxWorkers, pongsCount)
}
