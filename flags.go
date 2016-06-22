package main

import (
	"flag"
	"time"
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
	npackets   = flag.Int("c", 0, "Number of Packets")
	fDebug     = flag.Bool("d", false, "Set the SO_DEBUG option on the socket being used")
	fFlood     = flag.Bool("f", false, "Flood Ping")
	interval   = flag.Duration("i", time.Second, "Interval between pings in miliseconds")
	preload    = flag.Int("l", 0, "Preload")
	fNumeric   = flag.Bool("n", false, "Numeric output only")
	pattern    = flag.String("p", "0", "Data fill Pattern")
	fQuiet     = flag.Bool("q", false, "Quiet")
	fRroute    = flag.Bool("R", false, "RRoute")
	fDontRoute = flag.Bool("r", false, "Dont Route")
	fVerbose   = flag.Bool("v", false, "Verbose mode")
	datalen    = flag.Int("s", MAXPACKET, "Packet Size")
	mloopback  = flag.Bool("L", false, "Supress loopback of multicast packets")
	ttl        = flag.Int("T", 10, "Time to live for multicast packets")
	minterface = flag.String("I", "", "Interface")
)
