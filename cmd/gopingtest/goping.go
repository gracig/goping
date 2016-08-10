package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gracig/goping"
	"github.com/gracig/goping/pingers/icmpv4"
)

func main() {
	cfg := goping.Config{
		Count:      10,                             //a negative number will ping forever
		Interval:   time.Duration(1 * time.Second), //The interval between a host ping
		PacketSize: 100,                            //The packet size. It is nos implemented correctly yet. Now only local time are being send in the ping packet
		TOS:        0,                              //Type Of Sevice being passed. Only for linux and mac
		TTL:        64,                             //Time-To-Live, Only for Linux and Mac
		Timeout:    time.Duration(3 * time.Second), //The max time to wait for an answer
	}
	p := goping.New(cfg, icmpv4.New(), nil, nil)
	ping, pong, err := p.Start(time.Duration(1 * time.Millisecond))
	if err != nil {
		log.Fatal("Could not start pinger!")

	}
	go func() {
		for i := 0; i < 10; i++ { //Sending 10 jobs
			ping <- p.NewRequest("localhost", map[string]string{"job": strconv.Itoa(i)})
		}
		close(ping)
	}()
	for resp := range pong {
		fmt.Printf("Received response %v\n", resp)
	}
}
