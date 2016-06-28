package goping

import (
	"math/rand"
	"testing"
	"time"
	"sync"
)

func TestPauseResume(t *testing.T) {
	go Run()

	rand.Seed(1)
	var wg sync.WaitGroup
	wg.Add(200)
	go func() {
		for i := 0; i < 100; i++ {
			Pause()
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			wg.Done()
		}
	}()
	go func() {
		for i := 0; i < 100; i++ {
			Resume()
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			wg.Done()
		}
	}()
	wg.Wait()

}

func TestTimeout(t *testing.T){
	go Run()
	p:=Ping{Host:"localhost",Count:10}
	_ = p


}
