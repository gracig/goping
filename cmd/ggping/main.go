package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/gersongraciani/ggping"
)

const repeatGroup int = 1
const maxWorkers int = 100

//2015/10/15 00:15:05 Ping Unmatched Response: PING farts.com (184.168.164.84) 100(128) bytes of data.
//2015/10/15 00:15:05 Ping Unmatched Response: PING e10088.dspb.akamaiedge.net (23.213.202.147) 100(128) bytes of data.
//2015/10/15 00:15:05 Ping Unmatched Response: PING cs60.can.transactcdn.com (192.16.31.62) 100(128) bytes of data.
//2015/10/15 00:15:06 Ping Unmatched Response: PING www.gtm.nytimes.com (170.149.161.130) 100(128) bytes of data.
//2015/10/15 00:15:06 Ping Unmatched Response: PING e2094.b.akamaiedge.net (23.11.248.19) 100(128) bytes of data.
//2015/10/15 00:15:06 Ping Unmatched Response: PING www.google.com (173.194.205.103) 100(128) bytes of data.

func main() {
	requests := []*ggping.PingRequest{
		//````		{HostDest: "www.cnn.com", Tos: 16, Timeout: 1, MaxPings: 10, MinWait: 0.1, Percentil: 90, UserMap: map[string]string{"company": "cnn"}},
		{HostDest: "www.farts.com", Timeout: 1, MaxPings: 10, MinWait: 1, Percentil: 90, UserMap: map[string]string{"company": "farts"}},
		{HostDest: "www.microsoft.com", Timeout: 1, MaxPings: 10, MinWait: 1, Percentil: 90, UserMap: map[string]string{"company": "microsoft"}},
		{HostDest: "www.dell.com", Timeout: 1, MaxPings: 10, MinWait: 1, Percentil: 90, UserMap: map[string]string{"company": "dell"}},
		{HostDest: "www.nytimes.com", Timeout: 1, MaxPings: 10, MinWait: 1, Percentil: 90, UserMap: map[string]string{"company": "nytimes"}},
		{HostDest: "www.avaya.com", Timeout: 1, MaxPings: 10, MinWait: 1, Percentil: 90, UserMap: map[string]string{"company": "avaya"}},
		{HostDest: "www.google.com", Timeout: 1, MaxPings: 10, MinWait: 1, Percentil: 90, UserMap: map[string]string{"company": "google"}},
	}

	var wg sync.WaitGroup
	wg.Add(len(requests) * repeatGroup)
	done := make(chan *ggping.PingTask) //Creates the done channel to receive PingTasks as it arrives
	go func() {
		for task := range done {
			debug.Printf("\n\n%v %v", task.Request, task.Summary)
			for _, v := range task.Responses {
				debug.Printf("[%v]", v)
			}
			wg.Done()
		}
	}()
	startTime := time.Now()

	repeatedRequests := make([]*ggping.PingRequest, 0, len(requests)*repeatGroup)
	for i := 0; i < repeatGroup; i++ {
		for j, request := range requests {
			repeatedRequests = append(repeatedRequests, request)
			repeatedRequests[i].UserMap["request"] = fmt.Sprintf("%v", i+j)
		}
	}
	debug.Printf("Requests: %v, Workers: %v", len(repeatedRequests), maxWorkers)
	ggping.StartPing(done, repeatedRequests, maxWorkers)

	close(done) //Closes the done channel
	wg.Wait()
	debug.Printf("End of Program, elapsed time: %v workers:%v requests:%v", time.Since(startTime), maxWorkers, len(repeatedRequests))
}
