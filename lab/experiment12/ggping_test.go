package goping

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

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
	{Peak: 10, Iter: 1500, To: "ips", Timeout: 10 * time.Second, Interval: 1000 * time.Millisecond, TOS: 0, TTL: 64, REQUESTS: 100, PKTSZ: 1, UserMap: nil},
	//{Peak: 10, Iter: 1500, To: "192.168.0.1", Timeout: 4 * time.Second, Interval: 000 * time.Millisecond, TOS: 0, TTL: 64, REQUESTS: 100, PKTSZ: 10, UserMap: nil},
	/*
		{Peak: 0, Iter: 3, To: "www.google.com", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
		{Peak: 10, Iter: 3, To: "192.168.0.1", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
		{Peak: 10, Iter: 3, To: "www.terra.com.br", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
		{Peak: 10, Iter: 3, To: "www.uol.com.br", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
		{Peak: 10, Iter: 3, To: "www.ig.com.br", Timeout: 4 * time.Second, Interval: 1 * time.Second, TOS: 0, TTL: 64, REQUESTS: 10, PKTSZ: 100, UserMap: nil},
	*/
}

func TestPingOnChan(t *testing.T) {

	var wg sync.WaitGroup
	wg.Add(len(pingTaskTable))
	for j, t := range pingTaskTable {
		go func(t pingTaskRow, j int) {
			defer wg.Done()
			s := Settings{
				Bytes:          int(t.PKTSZ),
				CyclesCount:    t.REQUESTS,
				CyclesInterval: t.Interval,
				Timeout:        t.Timeout,
				TOS:            t.TOS,
				TTL:            t.TTL,
			}

			//Populating the list
			var targets []*Target
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
				targets = append(targets, &Target{IP: ip, UserData: t.UserMap})
			}
			if scanner.Err() != nil {
				log.Fatal("Error scanning ips file")
			}
			f.Close()
			/*
				for i := 0; i < t.Iter; i++ {
					ip, err := net.ResolveIPAddr("ip4", t.To)
					if err != nil {
						log.Println("Could not resolve address: ", t.To)
						continue
					}
					targets = append(targets, &Target{IP: ip, UserData: t.UserMap})
				}
			*/

			SetPoolSize(1000)
			chreply := RunCh(targets, s)

			for reply := range chreply {
				fmt.Printf("%v-%v\n", reply.Target.IP, reply)
			}
		}(t, j)
	}
	wg.Wait()

}
