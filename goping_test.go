package goping

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestPauseResume(t *testing.T) {
	Run()
	loop := 10

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

	Resume() //In case we are paused, this unblocks the main loop
}

func TestTimeoutWithTwoContexts(t *testing.T) {
	Run()
	loop := 1000

	ctx := NewContext()
	for i := 0; i < loop; i++ {
		ping := Ping{Host: "localhost", Count: 10, Timeout: 300, Data: make(map[string]string)}
		ping.Data["id"] = fmt.Sprintf("%5d", i)
		Add(ping, ctx)
	}

	ctx2 := NewContext()
	for i := 0; i < loop; i++ {
		ping := Ping{Host: "localhost", Count: 10, Timeout: 300, Data: make(map[string]string)}
		ping.Data["id2"] = fmt.Sprintf("%5d", i)
		Add(ping, ctx2)
	}

	done := make(chan struct{})
	go func() {
		var counter int
		for pp := range ctx.RecvChannel() {
			if pp.Done {
				counter++
			}
		}
		done <- struct{}{}
		if counter != loop {
			t.Errorf("Expecting %v responses but got %v", loop, counter)
		}
	}()

	done2 := make(chan struct{})
	go func() {
		var counter int
		for pp := range ctx2.RecvChannel() {
			if pp.Done {
				counter++
			}
		}
		done2 <- struct{}{}
		if counter != loop {
			t.Errorf("Expecting %v responses but got %v", loop, counter)
		}
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

}
