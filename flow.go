package ggping

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const minHelperWorkersFactor int = 10
const maxNumberOfWorkers int = 1000

func StartPing(done chan *PingTask, requests []*PingRequest, maxWorkers int) error {

	//The variable wg will be used as a semaphore to guarantee all summaries are done before the function exits
	var wg sync.WaitGroup

	//Parameters Validation
	if requests == nil || len(requests) < 1 {
		return fmt.Errorf("[]PingRequest should have 1 or more elements and not to be Nil")
	}

	//Defines the maximum number of worker as 1000.
	if !(maxWorkers >= 1 && maxWorkers <= maxNumberOfWorkers) {
		return fmt.Errorf("Max Workers should be between 1 and %v. Received value: %v", maxNumberOfWorkers, maxWorkers)
	}

	//Channel creation
	ping := make(chan *PingTask, maxWorkers*len(requests)*100) //Create a channel to send PingTask to ping (used by workerPings)   (ping)
	defer close(ping)
	pong := make(chan *PingTask) //Create a channel to receive PingTask after a Response is received (pong)
	defer close(pong)
	wait := make(chan *PingTask) //Create a channel to receive PingTask  and wait
	defer close(wait)
	summary := make(chan *PingTask) //Create a channel to send PingTask to summarize after all Responses are done (summary)
	defer close(summary)

	//Reads from the ping channel, call function that processes the ping and send the task to the pong channel
	var pingWorker = func(i int) {
		for pingTask := range ping {
			debug.Printf("Start Ping t%v %v\n", i, pingTask.Request)
			processPing(pingTask)
			pong <- pingTask //Send PingTask to pong channel
		}
	}
	//Reads from the pong channel, call function that processes the ping response.
	//The processSummary returns if the request has all the responses.
	//If all responses arrived then send to the summary channel.
	//Else send to the wait channel
	var pongWorker = func(i int) {
		for pingTask := range pong {
			debug.Printf("Start Pong t%v %v\n", i, pingTask.Request)
			if processPong(pingTask) {
				summary <- pingTask //Sends the pingTask to the summary channel
			} else {
				wait <- pingTask
			}
		}
	}

	//Reads from the wait channel
	//The ping request has a minWait time, that is the minimum ammount of seconds to wait before send another ping to the same ip.
	//This guarantes that the ping does not overloads devices.
	//After the sleep time, the task is sent to the ping channel.
	var waitWorker = func(i int) {
		for pingTask := range wait {
			debug.Printf("Sleeping t%v %v\n", i, pingTask.Request)
			//Sleep till the MinWait is reached for this Request type
			//pingTask.mutex.Lock()
			time.Sleep((time.Duration(pingTask.Request.MinWait) * time.Second) - pingTask.timeSinceLastResponse())
			//pingTask.mutex.Unlock()

			debug.Printf("Sending after wakeup t%v %v\n", i, pingTask.Request)
			ping <- pingTask //Sends the pingTask back to the ping channel

		}
	}

	//Reads from the summary channel.
	//Call the function processSummary to summarize all responses in a PingSummary Object and attach it to the Task
	//Send the Task to the done channel. The done channel was received as a parameter and is controlled by the caller
	var summaryWorker = func(i int) {
		for pingTask := range summary {
			debug.Printf("Start Summary t%v %v\n", i, pingTask.Request)
			processSummary(pingTask) //Summarizes all the responses inside the pingTask
			done <- pingTask         //Sends the job to the "done" channel
			wg.Done()                //Decrement wait group
		}
	}

	//Spawn Workers that read the "ping" channel
	for i := 0; i < maxWorkers; i++ {
		go pingWorker(i)

	}

	//Spawn Workers that read the "wait"channel. Always have minWaitWorkersFactor
	for i := 0; i < (maxWorkers + minHelperWorkersFactor); i++ {
		go waitWorker(i)
	}

	//Spawn other workers
	for i := 0; i < minHelperWorkersFactor; i++ {
		//Spawn Workers that read the "pong" channell
		go pongWorker(i)

		//Spawn Workers that read the  "summary" channel
		go summaryWorker(i)
	}

	//Dispatches all requests creating a PingTask for each request and sending it to the channel,
	//The worker that is listening to the chanell ping will process the request
	//Inside the PingTask object, the Responses slice was allocate with the request MaxPings parameter with length 0
	//The pong worker should then append responses to that as soon as it reads it from the pong channel
	for i, request := range requests {
		request.Number = i
		//Paramater Defaults
		if request.Payload == 0 {
			request.Payload = 100
		}
		if request.Timeout == 0 {
			request.Timeout = 3
		}
		if request.Tos == 0 {
			request.Tos = 0
		}
		if request.MinWait == 0 {
			request.MinWait = 1
		}
		pingTask := &PingTask{Request: request, Responses: make([]*PingResponse, 0, request.MaxPings)}
		ip, err := net.ResolveIPAddr("ip4", request.HostDest)
		if err != nil {
			pingTask.Error = err
			done <- pingTask //Sends the job to the "done" channel
			continue
		}
		request.IP = ip
		wg.Add(1)
		debug.Printf("Sending %v\n", request)
		ping <- pingTask
	}

	//Waits for all summaries be done, then exit function
	wg.Wait()

	return nil

}
