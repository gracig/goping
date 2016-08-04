package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gracig/goping"
	"github.com/gracig/goping/pingers/icmpv4"
)

const ()

var (
	help      bool
	hosts     []string
	smoothDur time.Duration = time.Duration(1 * time.Millisecond)
	cfg                     = goping.Config{
		Count:      -1,
		Interval:   time.Duration(1 * time.Second),
		PacketSize: 100,
		TOS:        0,
		TTL:        64,
		Timeout:    time.Duration(3 * time.Second),
	}
)

func parseFlags() {
	flag.BoolVar(&help, "help", false, "Display Help Message")
	flag.BoolVar(&help, "h", false, "Display Help Message")
	flag.IntVar(&cfg.Count, "count", -1, "The number of pings to send for each host")
	flag.IntVar(&cfg.Count, "c", -1, "The number of pings to send for each host")
	flag.DurationVar(&smoothDur, "smoothinterval", time.Duration(1*time.Millisecond), "The minimum interval time between all ping requests")
	flag.DurationVar(&smoothDur, "s", time.Duration(1*time.Millisecond), "The minimum interval time between all ping requests")
	flag.DurationVar(&cfg.Interval, "interval", time.Duration(1*time.Second), "The minimum interval time between pings from the same host")
	flag.DurationVar(&cfg.Interval, "i", time.Duration(1*time.Second), "The minimum interval time between pings from the same host")
	flag.DurationVar(&cfg.Timeout, "timeout", time.Duration(4*time.Second), "The maximum time to wait for a ping response")
	flag.DurationVar(&cfg.Timeout, "t", time.Duration(4*time.Second), "The maximum time to wait for a ping response")
	flag.IntVar(&cfg.PacketSize, "packetsize", 64, "The size of the ICMP Packet in every request")
	flag.IntVar(&cfg.PacketSize, "ps", 64, "The size of the ICMP Packet in every request")
	flag.IntVar(&cfg.TOS, "TOS", 0, "The TOS (Type of Service) field in the ip header")
	flag.IntVar(&cfg.TTL, "TTL", 64, "The TTL (Time to Live) field in the ip header")
	flag.Parse()
	hosts = flag.Args()
	if help || len(hosts) == 0 {
		flag.Usage()
		os.Exit(0)
	}
}

func main() {

	parseFlags()

	gp := goping.New(cfg, icmpv4.New(), nil, nil)

	ping, pong, err := gp.Start(smoothDur)
	if err != nil {
		log.Fatalf("Could not initialize pinger: %v", err)
	}

	go func() {
		for _, host := range hosts {
			ping <- gp.NewRequest(host, nil)
		}
		close(ping)
	}()

	//A map that count number of responses by each error found. nil = OK
	var counter = make(map[string]uint64)

	//Read all responses from the pong channel
	for r := range pong {
		printResponse(r)
		counter["TOTAL"]++
		if r.Err != nil {
			counter[r.Err.Error()]++
		} else {
			counter["OK"]++
		}

	}

	//Logging counter values
	var buf bytes.Buffer
	for k, v := range counter {
		buf.WriteString(fmt.Sprintf("%v=%v ", k, v))
	}
	log.Print(buf.String())
}
func printResponse(r goping.Response) {

	var msg = "%-7v %3d bytes from %-15v %-20v icmp_seq=%-5d ttl=%-2d tos=%-2d time(sec)=%-8.3f %-20v\n"
	if r.Err != nil {
		fmt.Printf(msg,
			"FAILURE",
			0, //0 bytes for packet size
			r.Peer,
			r.Request.Host,
			r.Seq,
			r.Request.Config.TTL,
			r.Request.Config.TOS,
			r.RTT,
			r.Err,
		)
	} else {
		fmt.Printf(msg,
			"SUCCESS",
			r.Request.Config.PacketSize, //0 bytes for packet size
			r.Peer,
			r.Request.Host,
			r.Seq,
			r.Request.Config.TTL,
			r.Request.Config.TOS,
			r.RTT,
			"", //No error will be shown
		)
	}

}
