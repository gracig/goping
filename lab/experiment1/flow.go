package goping

import (
	"time"
)

var (
	//	MaxWorker                 = os.Getenv("GGPING_MAX_WORKERS")
	//	MaxQueue                  = os.Getenv("GGPING_MAX_QUEUE")
	ping, chpong, wait, summary chan *Pong
	maxWorkers                  int  = 100
	isListening                 bool //This indicates if the Icmp Socket Listener is in Listening Mode
)

const (
	minHelperWorkersFactor int = 100
	maxNumberOfWorkers     int = 1000
)

func init() {
	//TODO: get MaxWorker from environment

	//Channel creation
	ping = make(chan *Pong, maxWorkers)     //Create a channel to send Pong to ping (used by workerPings)   (ping)
	chpong := make(chan *Pong, maxWorkers)  //Create a channel to receive Pong after a Response is received (pong)
	wait := make(chan *Pong, maxWorkers)    //Create a channel to receive Pong  and wait
	summary := make(chan *Pong, maxWorkers) //Create a channel to send Pong to summarize after all Responses are done (summary)

	//Reads from the ping channel, call function that processes the ping and send the pong to the pong channel
	var pingWorker = func(i int) {
		for pong := range ping {
			go func(pong *Pong) {
				debug.Printf("Start Ping t%v %v\n", i, pong.Request)
				processPing(pong)
				chpong <- pong //Send Pong to pong channel
			}(pong)
		}
	}
	//Reads from the pong channel, call function that processes the ping response.
	//The processSummary returns if the request has all the responses.
	//If all responses arrived then send to the summary channel.
	//Else send to the wait channel
	var pongWorker = func(i int) {
		for pong := range chpong {
			debug.Printf("Start Pong t%v %v\n", i, pong.Request)
			if processPong(pong) {
				go func(pong *Pong) {
					summary <- pong //Sends the pong to the summary channel
				}(pong)
			} else {
				go func(pong *Pong) {
					//debug.Printf("Sleeping t%v %v\n", i, pong.Request)
					//Sleep till the MinWait is reached for this Request type
					//time.Sleep((time.Duration(pong.Request.MinWait*1000) * time.Millisecond) - pong.timeSinceLastResponse())
					//debug.Printf("Sending after wakeup t%v %v\n", i, pong.Request)
					//				processPing(pong)
					//				chpong <- pong
					//ping <- pong //Sends the pong back to the ping channel
					wait <- pong
				}(pong)
			}
		}
	}

	//Reads from the wait channel
	//The ping request has a minWait time, that is the minimum ammount of seconds to wait before send another ping to the same ip.
	//This guarantes that the ping does not overloads devices.
	//After the sleep time, the pong is sent to the ping channel.
	var waitWorker = func(i int) {
		for pong := range wait {
			go func(pong *Pong) {
				debug.Printf("Sleeping t%v %v\n", i, pong.Request)
				//Sleep till the MinWait is reached for this Request type
				time.Sleep((time.Duration(pong.Request.MinWait*1000) * time.Millisecond) - pong.timeSinceLastResponse())
				debug.Printf("Sending after wakeup t%v %v\n", i, pong.Request)
				ping <- pong //Sends the pong back to the ping channel
			}(pong)

		}
	}

	//Reads from the summary channel.
	//Call the function processSummary to summarize all responses in a PingSummary Object and attach it to the Task
	//Send the Task to the done channel. The done channel was received as a parameter and is controlled by the caller
	var summaryWorker = func(i int) {
		for pong := range summary {
			debug.Printf("Start Summary t%v %v\n", i, pong.Request)
			processSummary(pong) //Summarizes all the responses inside the pong
			debug.Printf("Start Summary Processedt%v %v\n", i, pong.Request)
			go func(pong *Pong) {
				pong.Done <- true //Sends the job to the request "Pong" channel
				close(pong.Done)
			}(pong)
		}
	}

	for i := 0; i < maxWorkers; i++ {
		//Spawn Workers that read the "ping" channel
		go pingWorker(i)
		//Spawn Workers that read the "pong" channell
		go pongWorker(i)
		//Spawn Workers that read the  "summary" channel
		go summaryWorker(i)
	}

	//Spawn Workers that read the "wait"channel. Always have minWaitWorkersFactor
	for i := 0; i < (maxWorkers + minHelperWorkersFactor); i++ {
		go waitWorker(i)
	}

	//Spawn other workers
	//	for i := 0; i < minHelperWorkersFactor; i++ {
	//
	//	}
}
