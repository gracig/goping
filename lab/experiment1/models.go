package ggping

import (
	"fmt"
	"net"
	"sync"
	"time"
)

//The solely function of this library
//You call Ping passing a Ping Request as argument.
//i
//   request = &ggping.PingRequest{To:"localhost"} //Create a PingRequest object
//   ggping.Ping(request )  //Call the ping function
//   pong := <- request.Pong  //Receive the response using the channel Pong of the PingRequest object
//
//   Print Results:
//   if pong.Error != nill{
//	fmt.Println("Ping Error: %v",pong.Error)
//   }
//   fmt.Printf("Request %v", pong.Request)
//   fmt.Printf("summary %v", pong.Summary)
//   for _,v := range pong.Responses{
//	fmt.Println("Response %v",v)
//   }
var number int

func Ping(request *PingRequest) *Pong {
	//Default Parameters
	number++
	request.number = number
	if request.Payload == 0 {
		request.Payload = 52
	}
	if request.Timeout == 0 {
		request.Timeout = 3
	}
	if request.Tos == 0 {
		request.Tos = 0
	}
	pong := &Pong{Request: request, Responses: make([]*PingResponse, 0, request.MaxPings), Done: make(chan bool)}
	ip, err := net.ResolveIPAddr("ip4", request.To)
	if err != nil {
		go func() {
			pong.Error = fmt.Errorf("Could not resolve Ip address for: %v", request.To)
			pong.Done <- true //Sends the job back to the "Pong" channel and avoid uncessary pings
			close(pong.Done)
		}()
		return pong
	}
	request.ip = ip

	go func(pong *Pong) {
		debug.Printf("Sending %v\n", request)
		ping <- pong
	}(pong)

	return pong
}

//Struct that control the lifecycle of a Request, it holds the Request itself  and all of it PingResponses.
//This struct is mainly used for communication between channels and as the output for a user.
//Finally it holds the PingSummary that summarizes the PingResponses object
type Pong struct {
	Request   *PingRequest    //The user request received for internal use only
	Responses []*PingResponse //All the responses received
	Summary   *PingSummary    //The summary of all responses
	Error     error           //nil if success, error message if failed
	//Channel to receive the result as a Pong object. Take care with the circular reference on Pong->PingRequest
	Done chan bool //Channel where the Pong (Response) will be passed for the user
}

//Struct used to hold the options of a ping. It is sent to the PingWithPingRequest... methods
type PingRequest struct {
	//Attributes controlled  outside this package, unless it is created inside by the Ping function
	To        string            //FQDN or IP of the host to be pinged
	Timeout   float64           //Timeout in seconds
	Payload   float64           //Timeout in seconds
	Tos       uint              //Timeout in seconds
	MaxPings  int               //Max Number of pings that this request should do
	MinWait   float64           //The minimum in seconds time a ping should wait before do another ping
	UserMap   map[string]string //Store user defined metrics
	Percentil int               //Percentil to be used in the Summary

	//For Internal Use
	ip     *net.IPAddr //Converted IP Address of To. This conversion is made inside the package
	number int         // The Number of this request. This is controlled internally by the package
}

//Struct that holds each ping response status
type PingResponse struct {
	Rtt   float64   //Saves the round trip time from the ping in ms. should be 0 in case of error
	Error error     //nil if the ping succeeded, An error string otherwise
	When  time.Time //Time when the ping was received

	//For Internal Use
	mutex sync.Mutex //Control race conditions
}

//Struct that holds the summary of  the PingResponse objects
type PingSummary struct {
	PingsSent     int     //Ammount of ping commands sent to destination
	PingsReceived int     //Amount of successful pings returned
	PacketLoss    float64 //Percentage of Failed Pings related to sent Pings
	PacketSuccess float64 //Percentage of sucess Pings related to sent Pings
	RttAvg        float64 //Return the average RTT of all PingsReceived
	RttMax        float64 //Return the maximum RTT of all PingsReceived
	RttMin        float64 //Return the minimum RTT of all PingsReceived
	RttPerc       float64 //Return the xth percentil RTT. tha xth is received in the PingRequest object
	RttAvgPerc    float64 //Return the average RTT of all PingsReceived cutting values outside the xth percentil
	RttMaxPerc    float64 //Return the maximum RTT of all PingsReceived cutting values outside the xth percentil
	RttMinPerc    float64 //Return the minimum RTT of all PingsReceived cutting values outside the xth percentil
}

//Defines the interface for a pinger
//PingResponse should contain the Rtt, the error object if any, otherwise nil, the time.Now() value in the When attribute
type Pinger interface {
	Ping(*PingRequest, *PingResponse, int)
}
