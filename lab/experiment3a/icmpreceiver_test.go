package goping

import (
	"fmt"
	"net"
	"runtime"
	"time"
	//	"os"
	//	"os/exec"
	"testing"
	//	"time"

	//	"golang.org/x/net/icmp"
	//	"golang.org/x/net/ipv4"
)

func TestCoordinator(t *testing.T) {
	//fmt.Println(runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())
	//pings := 30000
	pings := 30000
	ping := make(chan Ping, pings*3)
	pong := make(chan Ping, pings*3)
	go pinger(ping, pong, (000 * time.Microsecond))
	p := Ping{To: "173.194.207.105", Timeout: 2}
	//p := Ping{To: "192.168.0.1", Timeout: 2}
	//p := Ping{To: "localhost", Timeout: 2}
	pl := Ping{To: "localhost", Timeout: 2}
	//Tries to convert the To attribute into an Ip attribute
	dst, _ := net.ResolveIPAddr("ip4", p.To)
	p.toaddr = dst
	dst, _ = net.ResolveIPAddr("ip4", pl.To)
	pl.toaddr = dst

	var max, min, sum float64

	//counter := pings
	go func() {
		for i := 0; i < pings; i++ {
			if i%1000 == 0 {
				ping <- pl
			} else {
				ping <- pl
			}
		}
	}()

	for i := 0; i < pings; i++ {
		reply := <-pong
		select {
		case t := <-reply.rttchan:
			close(reply.rttchan)
			rtt := float64(t.Sub(reply.When)) / float64(time.Millisecond)
			reply.Pong = Pong{Rtt: rtt}
			if reply.Pong.Rtt > max {
				max = reply.Pong.Rtt
			}
			if reply.Pong.Rtt < min || min == 0 {
				min = reply.Pong.Rtt
			}
			sum += reply.Pong.Rtt
			//		if reply.Pong.Rtt > 1 || reply.Pong.Rtt < 0 {
			fmt.Println(reply.When, reply.Seq, reply.To, reply.Pong.Rtt)
			//		}

		default:
			t := time.Now()
			if t.Sub(reply.When) > (time.Duration(reply.Timeout) * time.Second) {
				reply.Pong = Pong{Err: fmt.Errorf("Ping Response timed out")}
				fmt.Println(reply.When, t, reply.Seq, reply.Pong.Err)
			} else {
				//Put Reply back on channel, because it has not timed out yet
				i--
				pong <- reply
			}
		}
	}
	close(pong)
	fmt.Printf("Max: %v Min: %v Avg: %v\n", max, min, sum/float64(pings))
}

/*
func TestReceiver(t *testing.T) {
	fmt.Println("Starting Logger")
	messages := make(chan *rawIcmp, 100)
	var handler = func(r *rawIcmp) {
		messages <- r
	}

	fmt.Println("Running the Listener")
	go runListener(handler)
	fmt.Println("Starting Ping")
	cmd := exec.Command("ping", "-c10", "localhost")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()

	fmt.Println("Starting Select")
	mc := 0
LOOP:
	for {
		select {
		case ri := <-messages:
			rm, err := icmp.ParseMessage(1, ri.bytes[:ri.size])
			if err != nil {
				t.Errorf("Could not parse message")
			}
			if rm.Type != ipv4.ICMPTypeEchoReply {
				continue
			}
			body := rm.Body.(*icmp.Echo)
			if body.ID == cmd.Process.Pid {
				mc++
				fmt.Printf("Reply: %v %v %v %v %v %v %v\n", mc, rm, ri.cm.Src, ri.cm.Dst, body.ID, body.Seq, ri.when)
			}

		case <-time.After(time.Second * 2):
			break LOOP
		}
	}

	cmd.Wait()
	if mc != 10 {
		t.Errorf("Number of messages expteced were 10, received %v", mc)
	}
}
*/
