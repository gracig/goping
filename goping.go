package goping

import (
	"sync"
	"time"

	"github.com/gracig/goshared/log"
)

const (
	IPV4 = iota
	IPV6
	ICMP
	UDP
	TCP
	MAX_SEQUENCE = 65536
)

type Ping struct {
	Host string
}

type Pong struct {
	seq uint64
	RTT float64
}

func Add() {
}

var (
	chPing   chan Ping     = make(chan Ping)
	chPong   chan Pong     = make(chan Pong)
	chTicker *time.Ticker  = time.NewTicker(time.Second * 1)
	chPause  chan struct{} = make(chan struct{})
	chResume chan struct{} = make(chan struct{})
	chDone   chan struct{} = make(chan struct{})

	paused  bool = false
	pausedM sync.Mutex
)

func Run() {
	//Work with all states inside this function
	for {
		select {
		case <-chPing:
			log.Info.Printf("Called ping\n")
		case <-chPong:
			log.Info.Printf("Called pong\n")
		case <-chTicker.C:
			log.Info.Printf("Called ticker\n")
		case <-chPause:
			log.Info.Printf("Called Pause\n")
			<-chResume
			log.Info.Printf("Called Resume\n")
		case <-chDone:
			//Finalize and exit
		}
	}
}

func Pause() {
	pausedM.Lock()
	defer pausedM.Unlock()
	if !paused {
		chPause <- struct{}{}
		paused = true
		log.Info.Println("goping was succesfully paused")
	} else {
		log.Warn.Println("You requested to pause  an already paused goping")
	}
}

func Resume() {
	pausedM.Lock()
	defer pausedM.Unlock()
	if paused {
		chResume <- struct{}{}
		paused = false
		log.Info.Println("go ping was succesfully resumed")
	} else {
		log.Warn.Println("You requested to resume a non paused goping")
	}
}
