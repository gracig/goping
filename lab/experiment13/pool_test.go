package goping

import (
	"fmt"
	"math"
	"net"
	"runtime"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSingletonPool(t *testing.T) {
	Convey("Given an empty p Pooler var", t, func() {
		var p Pooler
		Convey("When populating p with the function Pool()", func() {
			p = Pool()
			Convey("The p value should not be nil", func() {
				So(p, ShouldNotBeNil)
			})
			Convey("When creating p2 from  Pool() singleton function", func() {
				p2 := Pool()
				Convey("p and p2 should be the same", func() {
					So(p, ShouldEqual, p2)
				})
			})
			Convey("When creating p3 from newPool() function", func() {
				p3 := newPool()
				Convey("p and p3 should not be the same", func() {
					So(p, ShouldNotEqual, p3)
				})
			})
		})
	})
}

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

func testNewRequestFromPoolAux(counter int) func() {

	var testGet = func(req Requester, treq trequest) {
		So(req.IPVersion(), ShouldEqual, treq.ipversion)
		So(req.IPProto(), ShouldEqual, treq.ipproto)
		So(req.Timeout(), ShouldEqual, treq.timeout)
		So(req.WhenSent().String(), ShouldEqual, treq.whenSent.String())
		//Messager interface
		So(req.Target(), ShouldEqual, treq.target)
		So(req.ICMPType(), ShouldEqual, treq.icmpType)
		So(req.ICMPCode(), ShouldEqual, treq.icmpCode)
		So(req.Size(), ShouldEqual, treq.size)
		So(req.TTL(), ShouldEqual, treq.ttl)
		So(req.TOS(), ShouldEqual, treq.tos)
	}
	var testSet = func(req Requester, treq trequest) {
		req.SetIPVersion(treq.ipversion)
		req.SetIPProto(treq.ipproto)
		req.SetTimeout(treq.timeout)
		req.SetWhenSent(treq.whenSent)
		req.SetTarget(treq.target)
		req.SetICMPType(treq.icmpType)
		req.SetICMPCode(treq.icmpCode)
		req.SetSize(treq.size)
		req.SetTTL(treq.ttl)
		req.SetTOS(treq.tos)
	}

	loopOverNewRequest := func() {
		pool := newPool()

		var mb, ma runtime.MemStats //The memStats variables to analyze memmory allocations
		runtime.GC()
		runtime.ReadMemStats(&mb) //Reading the before memory statistics

		var req Requester
		for i := 0; i < counter-1; i++ {
			req = pool.NewRequest()
			req.Release()
		}
		req = pool.NewRequest()
		defer req.Release()

		runtime.GC()
		runtime.ReadMemStats(&ma) //Reading the after memory statistics

		Convey("Then the number of memory allocations should be less than 5", func() {
			So(ma.Mallocs-mb.Mallocs-2, ShouldBeLessThan, 5)
			Convey("and req should not be nil", func() {
				So(req, ShouldNotBeNil)
				Convey(fmt.Sprintf("and req.Sequence() should return %v", counter%65536), func() {
					So(req.Sequence(), ShouldEqual, counter%65536)
					Convey("and req getter methods should return default values", func() {
						testGet(req, trequestMap["default"])
						Convey("and When we set non default values to req", func() {
							testSet(req, trequestMap["nondefault"])
							Convey("Then req getter methods  should return the same non default values", func() {
								testGet(req, trequestMap["nondefault"])
							})
						})
					})
				})
			})
		})
	}
	return loopOverNewRequest
}
func TestNewRequestFromPool(t *testing.T) {
	defer traceMem("TestNewRequestFromPool", statsNow())

	Convey("Given a function that returns a new pool ", t, FailureContinues, func() {
		Convey("When getting a new pool and calling req:=pool().NewRequest() and req.Release() 1 time", testNewRequestFromPoolAux(1))
		Convey("When getting a new pool and calling req:=pool().NewRequest() and req.Release() 65535 times", testNewRequestFromPoolAux(65535))
		Convey("When getting a new pool and calling req:=pool().NewRequest() and req.Release() 65536 times", testNewRequestFromPoolAux(65536))
		Convey("When getting a new pool and calling req:=pool().NewRequest() and req.Release() 120000 times", testNewRequestFromPoolAux(120000))
	})

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

func testNewResponseFromPoolAux(counter int) func() {
	var testGet = func(resp Responder, tresp tresponse) {
		if math.IsNaN(tresp.rtt) {
			So(math.IsNaN(resp.RTT()), ShouldEqual, math.IsNaN(tresp.rtt))
		} else {
			So(resp.RTT(), ShouldEqual, tresp.rtt)
		}
		So(resp.Request(), ShouldEqual, tresp.request)
		So(resp.WhenRecv().String(), ShouldEqual, tresp.whenRecv.String())
		So(resp.Error(), ShouldEqual, tresp.err)
		So(resp.Peer().String(), ShouldEqual, tresp.peer.String())

		//Messager interface
		So(resp.ICMPType(), ShouldEqual, tresp.icmpType)
		So(resp.ICMPCode(), ShouldEqual, tresp.icmpCode)
		So(resp.Size(), ShouldEqual, tresp.size)
		So(resp.TTL(), ShouldEqual, tresp.ttl)
		So(resp.TOS(), ShouldEqual, tresp.tos)
	}
	var testSet = func(resp Responder, tresp tresponse) {
		resp.SetRequest(tresp.request)
		resp.SetPeer(tresp.peer)
		resp.SetRTT(tresp.rtt)
		resp.SetWhenRecv(tresp.whenRecv)
		resp.SetError(tresp.err)

		resp.SetICMPType(tresp.icmpType)
		resp.SetICMPCode(tresp.icmpCode)
		resp.SetSize(tresp.size)
		resp.SetTTL(tresp.ttl)
		resp.SetTOS(tresp.tos)
	}

	loopOverNewResponse := func() {
		pool := newPool()

		var mb, ma runtime.MemStats //The memStats variables to analyze memmory allocations
		runtime.GC()
		runtime.ReadMemStats(&mb) //Reading the before memory statistics

		var resp Responder
		for i := 0; i < counter-1; i++ {
			resp = pool.NewResponse()
			resp.Release()
		}
		resp = pool.NewResponse()
		defer resp.Release()

		runtime.GC()
		runtime.ReadMemStats(&ma) //Reading the after memory statistics

		Convey("Then the number of memory allocations should be less than 5", func() {
			So(ma.Mallocs-mb.Mallocs-2, ShouldBeLessThan, 5)
			Convey("and resp should not be nil", func() {
				So(resp, ShouldNotBeNil)
				Convey(fmt.Sprintf("and resp.Sequence() should return %v", 0), func() {
					So(resp.Sequence(), ShouldEqual, 0)
					Convey("and resp getter methods should return default values", func() {
						testGet(resp, tresponseMap["default"])
						Convey("and When we set non default values to resp", func() {
							testSet(resp, tresponseMap["nondefault"])
							Convey("Then resp getter methods  should return the same non default values", func() {
								testGet(resp, tresponseMap["nondefault"])
							})
						})
					})
				})
			})
		})
	}
	return loopOverNewResponse
}
func TestNewResponseFromPool(t *testing.T) {
	defer traceMem("TestNewResponseFromPool", statsNow())

	Convey("Given a function that returns a new pool ", t, FailureContinues, func() {
		Convey("When getting a new pool and calling resp:=pool().NewResponse() and resp.Release() 1 time", testNewResponseFromPoolAux(1))
		Convey("When getting a new pool and calling resp:=pool().NewResponse() and resp.Release() 65535 times", testNewResponseFromPoolAux(65535))
		Convey("When getting a new pool and calling resp:=pool().NewResponse() and resp.Release() 65536 times", testNewResponseFromPoolAux(65536))
		Convey("When getting a new pool and calling resp:=pool().NewResponse() and resp.Release() 120000 times", testNewResponseFromPoolAux(120000))
	})

}
