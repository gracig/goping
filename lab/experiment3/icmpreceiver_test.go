package ggping

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func TestCoordinator(t *testing.T) {
	ping := make(chan Ping)
	pings := 3
	echochannel := make(chan *Ping, pings)

	go coordinator(ping, 2048)

	go func() {
		for i := 0; i < pings; i++ {
			ping <- Ping{To: "localhost", Timeout: 2, EchoChannel: echochannel}
		}
	}()
	for i := 0; i < pings; i++ {
		reply := <-echochannel
		fmt.Println(reply)
	}
	close(echochannel)

}
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
