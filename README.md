

# goping

Library to ping multiple hosts at once using go language in a controlled way. the interval between pings can be specified in nanosecond precision, but a good number would be 10 msec, with this interval one can send 100 pings/sec.

Please fell free to fix or contribute to the code. I was developing this alone, but life is very busy, and if anyone can contribute I would be grateful. This contribution may be done in build new kind of ping implementations or aprimorate the icmpv4 already here.

This is my first opensource project and my first attempt in Go. I studied the language specification, read lots of blogs, made several experimentations and tried to put what I have learned in this project.

The principal points about this code:

	- The code use the concept of Dependency Injection, We can inject other pings without interfere in the main logic.
	
	- The core library were tested using mocks (Because of the use of interfaces
	
	- gofmt goimports golint goracle were applied in the source code
	
	- Using channels is pretty cool, but is hard to find the "perfect" way. I think I find a good way to use channels in this library. But, well, It always exist a better way :-).
	
	- Make example applications to exercise the use of the library. Those are at tne cmd directory.

Basic Library Usage:

<pre>
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
			//Send ping requests to channel ping
			//A map with attributes can be passed in order to track the response if necessary.
			ping <- p.NewRequest("localhost", map[string]string{"job": strconv.Itoa(i)}) 
		}
		close(ping) //It  is very important to close the ping session after finishing sending ping, otherwise the program blocks.
	}()
	for resp := range pong { //Receiving ping responses from pong channels
		fmt.Printf("Received response %v\n", resp)
	}
}
</pre>


Known Issues: 

icmpv4_linux.go and icmpv4_darwin.go:

	-Need implement packetsize correctly. Now only the timestamp are being sent through the Data package.

	-Need implement ICMP Response Error message

	-Need implement a better RawMessage response.

icmpv4_windows.go

	-RTT time is not accurate as linux and darwinf pingers because windows does not implement the SO_TIMESTAMP flag. anyone has a 
	suggestion?
