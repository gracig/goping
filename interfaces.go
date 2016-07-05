package goping

import "net"

//NetResolver: Resolves a hostname to net.IP
type NetResolver interface {
	Resolve(net string, host string) (*net.IP, error)
}

type ICMPSender interface {
	//Send: Receive the Ping Request and send over the network
	//If any error, send a Poing object back describing the error
	Send(ping Ping)
}

type ICMPSenderFactory interface {
	GetICMPSender(ipVersion int, ipProto int)
}
