package goping

import (
	"fmt"
	"math"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

/*** Structures ***/

//Config is the configures a GoPing object
type Config struct {
	Count      int
	Interval   time.Duration
	Timeout    time.Duration
	TOS        int
	TTL        int
	PacketSize int
}

//Request represents a Ping Job. A request can generate 1 to Count responses
type Request struct {
	ID       uint64 //The unique id of this request
	Host     string
	Config   Config
	UserData map[string]string

	//Statistics
	Sent float64
}

//SeqRequest join a sequence field to be passed to unlderlying pingers
type SeqRequest struct {
	Seq int
	Req Request
}

//Response is sent for each Request Count iteration
type Response struct {
	Request Request
	RawResponse
}

//RawResponse represents the response send back by a Pinger
type RawResponse struct {
	Seq         int
	RTT         float64
	Peer        net.IP
	ICMPMessage []byte
	Err         error
}

/*** Interfaces ***/

//GoPinger coordinates ping requests and responses
type GoPinger interface {
	//NewRequest creates a new request object. Uses an id generator to populate the Id field
	NewRequest(hostname string, userData map[string]string) Request
	//Start initiates the request and response channels to which requests are sent and responses are received
	Start() (chan<- Request, <-chan Response, error)
}

/*** Interface Implementation ***/
type goping struct {
	cfg    Config
	pinger Pinger
	idGen  IDGenerator
	seqGen SequenceGenerator
}

//NewRequest creates a new request object. Uses an id generator to populate the Id field
func (g goping) NewRequest(hostname string, userData map[string]string) Request {
	return Request{
		ID:       g.idGen.Next(),
		Host:     hostname,
		Config:   g.cfg,
		UserData: userData,
	}
}

//Start initiates the request and response channels to which requests are sent and responses are received
func (g goping) Start() (chan<- Request, <-chan Response, error) {
	//Receives requests from the caller and send to "pin"
	in := make(chan Request)
	//Receives requests from "in" or "pin" and send to "pin" or "out"
	pin := make(chan Request)
	//Receives responses from "pin" . Caller consumes
	out := make(chan Response)
	//Receives signal from the caller and starts a goroutine that waits the pinger shutdown and send signal to done
	doneIn := make(chan struct{})
	//Receivex signal from goroutine inside doneIn and return the function
	done := make(chan struct{})
	//Used to wait for all ping requests to finish
	var wg sync.WaitGroup

	//Start the pinger channels
	ping, pong, pongdone, err := g.pinger.Start(os.Getpid())
	if err != nil {
		return nil, nil, fmt.Errorf("Could not start pinger: [%v]", err)
	}

	//Start the main loop in a goroutine.
	go func(in chan Request, out chan Response, ping chan<- SeqRequest, pong <-chan RawResponse, pongdone <-chan struct{}) {
		//This slice will hold the responses channels of a request.indexed by the icmp sequence number
		holder := make(map[int]chan RawResponse)
		//The main loop
		for {
			//Channel selection
			select {
			//Received a Request from Client
			case recv, open := <-in:
				//Verifies if channel is open
				if !open {
					//Stop reading from channel
					in = nil
					go func() {
						//Send signal to doneIn because channel is closed
						doneIn <- struct{}{}
					}()
				} else {
					//incrementing WaitGroup for later synchronization
					wg.Add(1)
					//Verifies if we have pings to do for this request based on the Count field
					if recv.Config.Count == 0 {
						//Request  Count is 0. Job is done without sending any requests
						wg.Done()
					} else {
						//Send request to be processed in a goroutine to not block this for loop
						go func() {
							pin <- recv
						}()
					}
				}
			//Received RawResponse from Pinger
			case rresp, open := <-pong:
				//Verifies if pong channel is open
				if open {
					//Verifies if holder has a respchan in the sequence nubuer provided by the response
					if holder[rresp.Seq] != nil {
						//Sends the raw response to the respchan.
						//It will be catch inside goroutine that waits for the response
						//The response channel has a buffer of size 1. So this will no block the for loop
						holder[rresp.Seq] <- rresp
						//Deletes the map entry
						delete(holder, rresp.Seq)
					}
				}
			//Received Request from in or pin
			case recv := <-pin:
				//Incrementing Request Sent Counter
				recv.Sent++
				//Create the SeqRequest struct
				sr := SeqRequest{Seq: g.seqGen.Next(recv.ID), Req: recv}
				//Creates a channel to receive the response
				respchan := make(chan RawResponse, 1)
				//Stores the channel in a slice indexed by the icmp sequence number
				holder[sr.Seq] = respchan
				//Send the request to the Pinger ping channel
				ping <- sr
				//Start a goroutinr to wait for the response
				go func(sr SeqRequest, respchan <-chan RawResponse) {
					//Schedule the wait interval for the next ping
					waitInterval := time.After(recv.Config.Interval)
					//Schedule the timeout while waiting for the response
					timeout := time.After(recv.Config.Timeout)
					//Builds the response object
					resp := Response{
						Request:     recv,
						RawResponse: RawResponse{Seq: sr.Seq, RTT: math.NaN()},
					}
					//Receive response or timeout
					select {
					case <-timeout:
						//Assign timeout error to response
						resp.Err = ErrTimeout
					case r := <-respchan:
						//Assign RawResponse to Response
						resp.RawResponse = r
					}
					//Send response to out channel. Blocks this function until the client consumes the response
					//We block because of the synchronization with waitgroup. Otherwise we would write to a closed channel.
					out <- resp
					//Verifies if we have more pings to do for this request
					if recv.Config.Count >= 0 && int(recv.Sent) >= recv.Config.Count {
						//This was the last last ping for this request. Job Done
						wg.Done()
					} else {
						//We still have more pings to do. Wait for the interval timer before send another request to pin channel
						<-waitInterval
						//Send another request to pin
						pin <- recv
					}
				}(sr, respchan)
			//Received signal that the "in" channel is closed. No more requests.
			case <-doneIn:
				//Starts a goroutine to wait for WaitGroup.
				go func(wg *sync.WaitGroup, out chan Response, done chan struct{}) {
					//Wait for all requests be finished including the timeouts
					wg.Wait()
					//Signals pinger that no more requests will be sent
					close(ping)
					//Waits pinger finish all tasks
					<-pongdone
					//Signals user that no more responses will be sent
					close(out)
					//Signals function to exit
					done <- struct{}{}
				}(&wg, out, done)
			//Received signal that we can exit the function
			case <-done:
				return
			}
		}
	}(in, out, ping, pong, pongdone)
	return in, out, nil
}

/*** New Methods ***/

//New creates a new gopinger object.
func New(cfg Config, pinger Pinger, seqGen SequenceGenerator, idGen IDGenerator) GoPinger {
	if seqGen == nil {
		seqGen = defaultSeqGen()
	}
	if idGen == nil {
		idGen = defaultIDGen()
	}
	return &goping{
		cfg:    cfg,
		pinger: pinger,
		seqGen: seqGen,
		idGen:  idGen,
	}
}

//SequenceGenerator returns a sequence number to be used in the ICMP sequence field.
type SequenceGenerator interface {
	Next(rid uint64) int
}
type seqGenerator struct {
	totalPings uint64
}

func (s *seqGenerator) Next(rid uint64) int {
	//Incrementing totalPings and getting sequence number
	return int(atomic.AddUint64(&(s.totalPings), 1) % 65536)
}

//IDGenerator return an id to a request
type IDGenerator interface {
	Next() uint64
}
type idGenerator struct {
	status uint64
}

func (i *idGenerator) Next() uint64 {
	return atomic.AddUint64(&(i.status), 1)
}

var defSeqGen *seqGenerator
var defSeqGenOnce sync.Once

func defaultSeqGen() SequenceGenerator {
	defSeqGenOnce.Do(func() { defSeqGen = new(seqGenerator) })
	return defSeqGen
}

var defIDGen *idGenerator
var defIDGenOnce sync.Once

func defaultIDGen() IDGenerator {
	defIDGenOnce.Do(func() { defIDGen = new(idGenerator) })
	return defIDGen
}
