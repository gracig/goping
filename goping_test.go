package goping

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestPauseResume(t *testing.T) {
	go Run()
	loop := 1000

	rand.Seed(1)
	var wg sync.WaitGroup
	wg.Add(loop)
	go func() {
		for i := 0; i < loop/2; i++ {
			Pause()
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			wg.Done()
		}
	}()
	go func() {
		for i := 0; i < loop/2; i++ {
			Resume()
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			wg.Done()
		}
	}()

	wg.Wait()

}

func TestTimeoutWithTwoContexts(t *testing.T) {
	go Run()

	ctx := NewContext()
	for i := 0; i < 1000; i++ {
		ping := Ping{Host: "localhost", Count: 10, Timeout: 3000, Data: make(map[string]string)}
		ping.Data["id"] = fmt.Sprintf("%5d", i)
		Add(ping, ctx)
	}

	ctx2 := NewContext()
	for i := 0; i < 1000; i++ {
		ping := Ping{Host: "localhost", Count: 10, Timeout: 3000, Data: make(map[string]string)}
		ping.Data["id2"] = fmt.Sprintf("%5d", i)
		Add(ping, ctx2)
	}

	done := make(chan struct{})
	go func() {
		for pp := range ctx.PingPongChannel() {
			if pp.Done {
				t.Logf("Received pp %v %v\n", pp, pp.Data)
			}
		}
		done <- struct{}{}
	}()

	done2 := make(chan struct{})
	go func() {
		for pp := range ctx2.PingPongChannel() {
			if pp.Done {
				t.Logf("Received pp %v %v\n", pp, pp.Data)
			}
		}
		done2 <- struct{}{}
	}()

	timer := time.NewTimer(time.Second * 60)
	select {
	case <-done:
		select {
		case <-done2:
		case <-timer.C:
			t.Error("Timeout took too long, something is wrong")
		}
	case <-timer.C:
		t.Error("Timeout took too long, something is wrong")
	}

	t.Logf("Pings Sent: %d", seq)

}
