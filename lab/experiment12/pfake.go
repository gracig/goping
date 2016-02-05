package goping

import (
	"sync"
	"time"
)

type icmpFake struct {
	chping chan *pinger
	wginit sync.WaitGroup
}

//Implements osping
func (this *icmpFake) init() {
	this.chping = make(chan *pinger)
	this.wginit.Add(1)
	go this.icmpSender()
	this.wginit.Wait()
}

//Implements osping
func (this *icmpFake) ping(p *pinger) time.Time {
	this.chping <- p
	return <-p.chWhen
}

//Send icmp Requests over a raw socket
func (this *icmpFake) icmpSender() {

	this.wginit.Done()

	for p := range this.chping {
		p.chWhen <- time.Now()
	}
}
