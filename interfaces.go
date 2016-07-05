package goping

import (
	"net"

	"golang.org/x/net/icmp"
)

type ICMPRequest interface {
	//Protocol: The ip protocol to send the ip message  (ICMP=1, TCP=6, UDP=17)
	Protocol() uint

	//IP: The target address of this request
	IP() net.IP

	//The ICMP Message to send
	Message() icmp.Message
}

type ICMPReply interface {
}
