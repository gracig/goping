package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gracig/goping"
	"github.com/gracig/goping/pingers/icmpv4"
)

func main() {
	cfg := goping.Config{
		Count:      -1,
		Interval:   time.Duration(1 * time.Second),
		PacketSize: 100,
		TOS:        0,
		TTL:        64,
		Timeout:    time.Duration(3 * time.Second),
	}
	p := goping.New(cfg, icmpv4.New(), nil, nil)
	ping, pong, err := p.Start()
	if err != nil {
		log.Fatal("Could not start pinger!")

	}
	go func() {
		for i := 0; i < 1; i++ {
			ping <- p.NewRequest("localhost", nil)
		}
		close(ping)
	}()
	for resp := range pong {
		fmt.Printf("Received response %v\n", resp)
	}
}
