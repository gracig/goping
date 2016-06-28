package goping

import (
	"math/rand"
	"testing"
	"time"
)

func TestPauseResume(t *testing.T) {
	go Run()

	rand.Seed(1)
	go func() {
		for i := 0; i < 100; i++ {
			Pause()
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
		}
	}()
	go func() {
		for i := 0; i < 100; i++ {
			Resume()
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
		}
	}()

	time.Sleep(10 * time.Second)
}
