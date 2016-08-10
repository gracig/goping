

# goping
Library to ping multiple hosts at once using go language
It works ok on Linux and Darwin because the elapsed time is retrieved the SO_TIMESTAMP flag and this flag is not implemented on windows.
The elapsed time on Windows is retrieved after retrieve the packet. May have a delay if too many pings are occuring at the same time.

An example program exists at github.com/gracig/goping/cmd/gopingtest

A command line program that mimics the ping utility exists at github.com/gracig/goping/cmd/goping

Basic Library Usage:
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
	
	p := goping.New(cfg, icmpv4.New(), nil, nil)  //Creates a new instance. Injecting the Pinger icmpv4. Using defaults for last two parameters.
	
	ping, pong, err := p.Start(time.Duration(1 * time.Millisecond)) //Initiates a session. ping and pong are two channels.
	
	if err != nil {
		log.Fatal("Could not start pinger!")

	}
	go func() {
		for i := 0; i < 10; i++ { //Sending 10 jobs
			ping <- p.NewRequest("localhost", map[string]string{"job": strconv.Itoa(i)}) //Send ping requests to channel ping
		}
		close(ping) //It  is very important to close the ping session after finishing sending ping, otherwise the program blocks.
	}()
	for resp := range pong { //Receiving ping responses from pong channels
		fmt.Printf("Received response %v\n", resp)
	}
}
