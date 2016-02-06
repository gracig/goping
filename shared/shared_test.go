package goping

import (
	"fmt"
	"math"
	"net"
	"time"
)

type trequest struct {
	target    Target
	sequence  int
	icmpType  byte
	icmpCode  byte
	size      int
	ttl       int
	tos       int
	pool      Pooler
	ipproto   uint8
	ipversion uint8
	timeout   time.Duration
	whenSent  time.Time
}
type target struct {
	Target
}

var trequestMap = map[string]trequest{
	"nondefault": {
		target:    new(target),
		icmpType:  15,
		icmpCode:  10,
		size:      23,
		ttl:       80,
		tos:       16,
		ipproto:   IPProtoUDP,
		ipversion: IPV6,
		timeout:   time.Millisecond,
		whenSent:  time.Now(),
	},
	"default": {
		target:    nil,
		icmpType:  8,
		icmpCode:  0,
		size:      56,
		ttl:       60,
		tos:       0,
		ipproto:   IPProtoICMP,
		ipversion: IPV4,
		timeout:   3 * time.Second,
		whenSent:  time.Time{},
	},
}

type tresponse struct {
	request  Requester
	peer     net.IP
	sequence int
	icmpType byte
	icmpCode byte
	size     int
	ttl      int
	tos      int
	pool     Pooler
	err      error
	rtt      float64
	whenRecv time.Time
}
type requester struct {
	Requester
}

var tresponseMap = map[string]tresponse{
	"nondefault": {
		request:  new(requester),
		peer:     net.IPv4(127, 0, 0, 1),
		icmpType: 15,
		icmpCode: 10,
		size:     23,
		ttl:      80,
		tos:      16,

		err:      fmt.Errorf("Test Error"),
		rtt:      1,
		whenRecv: time.Now(),
	},
	"default": {
		request:  nil,
		peer:     net.IPv4zero,
		icmpType: 0,
		icmpCode: 0,
		size:     56,
		ttl:      60,
		tos:      0,

		err:      nil,
		rtt:      math.NaN(),
		whenRecv: time.Time{},
	},
}
