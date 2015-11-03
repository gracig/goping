package ggping

import (
	"sync"
	"time"
)

const ()

var (
	seq   int        //The sequence number to be used when send ping packets
	seqmu sync.Mutex //The mutex to increment seq

	seqmap map[string]chan Pong
)

//The object that will store global ping parameters and perform the ping operations
//This should be invoked before adding ping requests
type Pinger struct {
	Timeout        float32      //The timeout, in secs,  to wait for a EchoReply
	Interval       float32      //The time to wait between pings on this host
	Iterations     int          //The number of messages to be sent to the target
	PacketSize     int          //The size of the Packet to be sent
	PongChannel    chan Pong    //A pointer to a channel of Pong objects. May be nil
	SummaryChannel chan Summary //A pointer to a channel of Summary objects. May be nil

	wg sync.WaitGroup //Indicates the end of the ping operations
}

//Synchronous Ping
func (p *Pinger) Ping(r Request) (*Pong, error) {
	if err := p.prepareIcmpMessage(&r); err != nil {
		return &Pong{Request: &r, When: time.Now(), Err: err}, err
	}
	return nil, nil
}

//Resolve the ip address if r.ip is nil
//Build the icmp message if r.msg is nil
//Set r.seq
func (p *Pinger) prepareIcmpMessage(r *Request) error {

	if r.ip == nil {
		//TODO: Set resolved ip. return error if couldn't
	}

	if r.msg == nil {
		//TODO: Build the icmp packet. Return error if couldn't
	}

	//Increments the sequence variable and assign to r.seq
	seqmu.Lock()
	defer seqmu.Unlock()
	seq = (seq + 1) % 65536
	r.seq = seq

	return nil
}
