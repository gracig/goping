package goping

import "net"

//NetResolver: Resolves a hostname to net.IP
type NetResolver interface {
	Resolve(net string, host string) (*net.IP, error)
}

type Config struct {
	Timeout   int
	Interval  int
	Count     int
	PcktSize  int
	Data      map[string]string
	IPVersion int
	IPProto   int
}

func NewPingRequest(nr NetResolver, host string, cfg Config) (Ping, error) {
	return Ping{}, nil
}

type ICMPSender interface {
	//Send: Receive the Ping Request and send over the network
	//If any error, send a Poing object back describing the error
	Send(ping Ping)
}
type ICMPSenderFactory interface {
	GetICMPSender(ipVersion int, ipProto int)
}

//GoPinger is the interface that
type Pinger interface {
	//Ping sends the Ping struct to the mainLoop through the chPing channel
	Send(p Ping)
	//Pong receives responses from the mainLoop
	PongChan() <-chan Response
	//Wait for all ping responses arrive
	Wait()
}

//New Creates a new Pinger.
func New(icmpSenderFactory ICMPSenderFactory) Pinger {
	return &pinger{
		Pongch: make(chan Response),
		pongch: make(chan Response),
	}
}
