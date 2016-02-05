package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/gersongraciani/goping/lab/experiment12"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile to this file")

type pingTaskRow struct {
	Peak     int
	Iter     int
	To       string
	Timeout  time.Duration
	Interval time.Duration
	TOS      int
	TTL      int
	REQUESTS int
	PKTSZ    uint
	UserMap  map[string]string
}

var pingTaskTable = []pingTaskRow{
	{Peak: 10, Iter: 1500, To: "ips.africa", Timeout: 20 * time.Second, Interval: 1000 * time.Millisecond, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 10, UserMap: nil},
	//{Peak: 10, Iter: 1500, To: "192.168.0.1", Timeout: 4 * time.Second, Interval: 000 * time.Millisecond, TOS: 0, TTL: 64, REQUESTS: 100, PKTSZ: 10, UserMap: nil},
	/*
		{Peak: 0, Iter: 3, To: "www.google.com", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
		{Peak: 10, Iter: 3, To: "192.168.0.1", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
		{Peak: 10, Iter: 3, To: "www.terra.com.br", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
		{Peak: 10, Iter: 3, To: "www.uol.com.br", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
		{Peak: 10, Iter: 3, To: "www.ig.com.br", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
	*/
}

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	var wg sync.WaitGroup
	wg.Add(len(pingTaskTable))
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal("Could not get hostname")
	}
	sourceIp, err := net.ResolveIPAddr("ip4", hostname)
	if err != nil {
		log.Fatal("Could not get hostname address")
	}
	for _, t := range pingTaskTable {
		s := goping.Settings{
			Bytes:          int(t.PKTSZ),
			CyclesCount:    t.REQUESTS,
			CyclesInterval: t.Interval,
			Timeout:        t.Timeout,
			TOS:            t.TOS,
			TTL:            t.TTL,
		}

		//Populating the list
		var targets []*goping.Target
		f, err := os.Open(t.To)
		if err != nil {
			log.Fatal("Could not open ips file")
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			host := scanner.Text()
			ip, err := net.ResolveIPAddr("ip4", host)
			if err != nil {
				log.Printf("Could not resolve address: %s\n", host)
				continue
			}
			targets = append(targets, &goping.Target{From: sourceIp, IP: ip, UserData: t.UserMap})
		}
		if scanner.Err() != nil {
			log.Fatal("Error scanning ips file")
		}
		f.Close()

		//goping.SetPoolSize(len(targets) * 3)
		goping.SetPoolSize(33000)
		chreply := goping.RunCh(targets, s)

		for reply := range chreply {
			fmt.Printf("%v-%v\n", reply.Target.IP, reply)
		}
		wg.Done()
	}
	wg.Wait()

}
