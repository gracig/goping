package goping

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestPauseResume(t *testing.T) {
	loop := 1000
	go Run()

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

func TestTimeout(t *testing.T) {
	go Run()
	ping := Ping{Host: "localhost", Count: 10, Timeout: 3000}
	ctx := NewContext()
	for i := 0; i < 100; i++ {
		Add(ping, ctx)
	}

	done := make(chan struct{})

	go func() {
		for pp := range ctx.PingPongChannel() {
			t.Logf("Received pp %v\n", pp)
		}
		done <- struct{}{}
	}()

	timer := time.NewTimer(time.Second * 60)
	select {
	case <-done:
	case <-timer.C:
		t.Error("Timeout took too long, something is wrong")
	}

}
