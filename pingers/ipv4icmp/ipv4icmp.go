package goping

import (
	"fmt"
	"net"
	"sync"
)

//Pinger is responsible for send and receive pings over the network
//type Pinger interface {
//	Ping(r Request) (future <-chan RawResponse, seq int, err error)
//}

type IPv4ICMPPinger interface {
	Pinger
}

type ipv4IcmpPinger struct {
	counter uint64
	ping    chan []byte
}

var iv4ip *ipv4IcmpPinger
var iv4ipOnce sync.Once

func GetIPv4ICMPPinger() IPv4ICMPPinger {
	if iv4ip == nil {
		iv4ipOnce.Do(func() {
			iv4ip = &ipv4IcmpPinger{}
		})
	}
	return iv4ip
}

func (p *ipv4IcmpPinger) Ping(r Request) (future <-chan RawResponse, seq int, err error) {

	//Get Sequence Number
	p.counter++
	seq = int(p.counter % 65536)

	//Get Future Channel
	fut := make(chan RawResponse, 1)
	future = fut

	//Resolve HostName
	if addr, e := net.ResolveIPAddr("ip4", r.Host); e != nil {
		err = fmt.Errorf("Could not resolve address: %s", r.Host)
	} else {
		_ = addr
	}

	return
}
