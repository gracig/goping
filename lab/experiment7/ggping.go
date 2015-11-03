package ggping

import (
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"
)

//Represents a Ping Request
type Request struct {
	//External variables
	To       string            //THe ip or FQDN to use as the icmp target
	UserData map[string]string //A map to hold user data

	//Internal variables
	ip    *net.IPAddr   //The resolved ip address to use when sending the ICMP Message. This must be filled at the first ping and be reused
	itPos int           //THe iterator position. When equals to Iterations, all the pings were sent. Starts at 1
	when  time.Time     //The time when the last packt was sent
	seq   int           //The icmp sequence to use when sending the ICMP Message.
	msg   *icmp.Message //Message to send to the target. It is built at the first ping. Other pings will reuse that, only incrementing the Seq field
}

//Represents a Ping response
type Pong struct {
	Request *Request  //THe original request object
	When    time.Time //The time the reply was received
	Err     error     // If eny error happened, this goes here, including timeout
	Rtt     float64   // THe elapsed time, in miliseconds, spending from throwing the packet and receiving the packet

	Peer net.Addr //THe peer that respond to the pong request
	Size int      //THe size of the packet
	Data []byte   //The bytes received from the network
}

//Summarization of all Responses from a Request
type Summary struct {
	Request *Request //The pointer for the Request object. So the user can reference that
	Replies []Pong   //A slice of EchoReply. The size will be the iterator Position
}
