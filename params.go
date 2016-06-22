package main

import (
	"flag"
	"os"
	"time"

	"github.com/gracig/goshared/log"
)

const (
	MAXPACKET = 64 - 8
)

/*
SYNOPSIS
     ping [-AaCDdfnoQqRrv] [-b boundif] [-c count] [-G sweepmaxsize]
     [-g sweepminsize] [-h sweepincrsize] [-i wait] [-k trafficclass]
     [-l preload] [-M mask | time] [-m ttl] [-P policy] [-p pattern]
     [-S src_addr] [-s packetsize] [-t timeout] [-W waittime] [-z tos] host

     ping [-AaDdfLnoQqRrv] [-b boundif] [-c count] [-I iface] [-i wait]
     [-k trafficclass] [-l preload] [-M mask | time] [-m ttl] [-P policy]
     [-p pattern] [-S src_addr] [-s packetsize] [-T ttl] [-t timeout]
     [-W waittime] [-z tos] mcast-group
*/
var (
	npackets   int
	fDebug     bool
	fFlood     bool
	interval   time.Duration
	preload    int
	fNumeric   bool
	pattern    string
	fQuiet     bool
	fRroute    bool
	fDontRoute bool
	fVerbose   bool
	datalen    int
	mloopback  bool
	ttl        int
	minterface string
)

func init() {
	var help bool
	flag.BoolVar(&help, "help", false, "Display Help Message")
	flag.BoolVar(&help, "h", false, "Display Help Message")
	flag.IntVar(&npackets, "c", 0, "Number of Packets")
	flag.BoolVar(&fDebug, "d", false, "Set the SO_DEBUG option on the socket being used")
	flag.BoolVar(&fFlood, "f", false, "Flood Ping")
	flag.DurationVar(&interval, "i", time.Second, "Interval between pings in miliseconds")
	flag.IntVar(&preload, "l", 0, "Preload")
	flag.BoolVar(&fNumeric, "n", false, "Numeric output only")
	flag.StringVar(&pattern, "p", "0", "Data fill Pattern")
	flag.BoolVar(&fQuiet, "q", false, "Quiet")
	flag.BoolVar(&fRroute, "R", false, "RRoute")
	flag.BoolVar(&fDontRoute, "r", false, "Dont Route")
	flag.BoolVar(&fVerbose, "v", false, "Verbose mode")
	flag.IntVar(&datalen, "s", MAXPACKET, "Packet Size")
	flag.BoolVar(&mloopback, "L", false, "Supress loopback of multicast packets")
	flag.IntVar(&ttl, "T", 10, "Time to live for multicast packets")
	flag.StringVar(&minterface, "I", "", "Interface")

	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}

	log.Debug.Printf("Parameter %v=%v\n", "npackets", npackets)
	log.Debug.Printf("Parameter %v=%v\n", "fDebug", fDebug)
	log.Debug.Printf("Parameter %v=%v\n", "fFlood", fFlood)
	log.Debug.Printf("Parameter %v=%v\n", "interval", interval)
	log.Debug.Printf("Parameter %v=%v\n", "preload", preload)
	log.Debug.Printf("Parameter %v=%v\n", "fNumeric", fNumeric)
	log.Debug.Printf("Parameter %v=%v\n", "pattern", pattern)
	log.Debug.Printf("Parameter %v=%v\n", "fQuiet", fQuiet)
	log.Debug.Printf("Parameter %v=%v\n", "fRroute", fRroute)
	log.Debug.Printf("Parameter %v=%v\n", "fDontRoute", fDontRoute)
	log.Debug.Printf("Parameter %v=%v\n", "fVerbose", fVerbose)
	log.Debug.Printf("Parameter %v=%v\n", "datalen", datalen)
	log.Debug.Printf("Parameter %v=%v\n", "mloopback", mloopback)
	log.Debug.Printf("Parameter %v=%v\n", "ttl", ttl)
	log.Debug.Printf("Parameter %v=%v\n", "minterface", minterface)

}
