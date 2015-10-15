package ggping

import (
	"net"
	"sync"
	"time"
)

//START: Model Definition
//Defines the interface for a Pinger.
//At this moment we will create a linuxpinger only to use the ping command from the OS
//Later we would implement other pings. The final solution would be a Go Pinger with no dependencies
type PingRequest struct {
	IP        *net.IPAddr       //Converted IP Address of HostDest
	Number    uint              // The Number of this request.
	HostDest  string            //FQDN or IP of the host to be pinged
	Timeout   float64           //Timeout in seconds
	Payload   float64           //Timeout in seconds
	Tos       uint              //Timeout in seconds
	MaxPings  int               //Max Number of pings that this request should do
	MinWait   float64           //THe minimum in seconds time a ping should wait before do another ping
	UserMap   map[string]string //Store user defined metrics
	Percentil int               //Percentil to be used in the Summary session
}

type PingResponse struct {
	Rtt   float64   //Saves the round trip time from the ping in ms. should be 0 in case of error
	Error error     //nil if the ping succeeded, An error string otherwise
	When  time.Time //Time when the ping was received
}

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

//Structure to be passed accross the channels. Channels will have all information related to the request,responses and summaries
type PingTask struct {
	Request   *PingRequest    //The user request received
	Responses []*PingResponse //All the responses received
	Summary   *PingSummary    //The summary of all responses
	Error     error           //nil if success, error message if failed
	mutex     sync.Mutex      //Control race conditions
}

func (t PingTask) timeSinceLastResponse() time.Duration {
	lastResponse := t.Responses[len(t.Responses)-1]
	return time.Since(lastResponse.When)
}

//Defines the interface for a pinger
//PingResponse should contain the Rtt, the error object if any, otherwise nil, the time.Now() value in the When attribute
type Pinger interface {
	Ping(*PingRequest, *PingResponse, int)
}
