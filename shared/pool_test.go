package goping

import (
	"math"
	"runtime"
	"testing"
)

func TestSingletonPool(t *testing.T) {

	pa := Pool()
	if pa == nil {
		t.Error("Expected p not nil", pa)
	}

	pb := Pool()
	if pa != pb {
		t.Error("Expected Pool() to return same pointer values per call")
	}

	pc := newPool()
	if pc == pa {
		t.Error("Expected newPool() return different values per call")
	}
}

func TestNewRequestFromPool(t *testing.T) {
	defer traceMem("TestNewRequestFromPool", statsNow())

	//Calling for 1 time
	testNewRequestFromPoolAux(t, 1)

	//Calling for 65535 times
	testNewRequestFromPoolAux(t, 65535)

	//Calling for 65536 times . Tests the Sequence counter reset. since its size is 16 bits
	testNewRequestFromPoolAux(t, 65536)

	//Calling for 120000 times . Tests the Sequence counter reset. since its size is 16 bits
	testNewRequestFromPoolAux(t, 120000)

}
func TestNewResponseFromPool(t *testing.T) {
	defer traceMem("TestNewResponseFromPool", statsNow())

	//Calling for 1 time
	testNewResponseFromPoolAux(t, 1)

	//Calling for 65535 times
	testNewResponseFromPoolAux(t, 65535)

	//Calling for 65536 times . Tests the Sequence counter reset. since its size is 16 bits
	testNewResponseFromPoolAux(t, 65536)

	//Calling for 120000 times . Tests the Sequence counter reset. since its size is 16 bits
	testNewResponseFromPoolAux(t, 120000)

}

func testNewResponseFromPoolAux(t *testing.T, counter int) func() {
	var testGet = func(resp Responder, tresp tresponse) {
		if math.IsNaN(tresp.rtt) {
			if math.IsNaN(resp.RTT()) != math.IsNaN(tresp.rtt) {
				t.Error("Expected ", math.IsNaN(resp.RTT()), ". Got: ", math.IsNaN(tresp.rtt))
			}
		} else {
			if resp.RTT() != tresp.rtt {
				t.Error("Expected:", resp.RTT(), " Got:", tresp.rtt)
			}
		}
		if resp.Request() != tresp.request {
			t.Error("Expected:", resp.Request(), " Got:", tresp.request)
		}
		if resp.WhenRecv().String() != tresp.whenRecv.String() {
			t.Error("Expected:", resp.WhenRecv().String(), " Got:", tresp.whenRecv.String())
		}
		if resp.Error() != tresp.err {
			t.Error("Expected:", resp.Error(), " Got:", tresp.err)
		}
		if resp.Peer().String() != tresp.peer.String() {
			t.Error("Expected:", resp.Peer().String(), " Got:", tresp.peer.String())
		}

		if resp.ICMPType() != tresp.icmpType {
			t.Error("Expected equal values: ", resp.ICMPType(), tresp.icmpType)
		}
		if resp.ICMPCode() != tresp.icmpCode {
			t.Error("Expected equal values: ", resp.ICMPCode(), tresp.icmpCode)
		}
		if resp.Size() != tresp.size {
			t.Error("Expected equal values: ", resp.Size(), tresp.size)
		}
		if resp.TTL() != tresp.ttl {
			t.Error("Expected equal values: ", resp.TTL(), tresp.ttl)
		}
		if resp.TOS() != tresp.tos {
			t.Error("Expected equal values: ", resp.TOS(), tresp.tos)
		}

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

		if (ma.Mallocs - mb.Mallocs - 2) >= 5 {
			t.Error("Expected number of allocations be less than 5", (ma.Mallocs - mb.Mallocs - 2))
		}
		if resp == nil {
			t.Error("Expected resp to be not nil")
		}
		if resp.Sequence() != (counter % 65536) {
			t.Error("Expected Sequence number to be ", (counter % 65536), ". got: ", resp.Sequence())
		}
		//Testing Getting Default Values
		testGet(resp, tresponseMap["default"])

		//Testing Set Non Default Values
		testSet(resp, tresponseMap["nondefault"])

		//Testing Getting Non Default Values
		testGet(resp, tresponseMap["nondefault"])

	}
	return loopOverNewResponse
}

func testNewRequestFromPoolAux(t *testing.T, counter int) func() {

	var testGet = func(req Requester, treq trequest) {
		if req.IPVersion() != treq.ipversion {
			t.Error("Expected equal values: ", req.IPVersion(), treq.ipversion)
		}
		if req.IPProto() != treq.ipproto {
			t.Error("Expected equal values: ", req.IPProto(), treq.ipproto)
		}
		if req.Timeout() != treq.timeout {
			t.Error("Expected equal values: ", req.Timeout(), treq.timeout)
		}
		if req.WhenSent().String() != treq.whenSent.String() {
			t.Error("Expected equal values: ", req.WhenSent(), treq.whenSent)
		}

		if req.Target() != treq.target {
			t.Error("Expected equal values: ", req.Target(), treq.target)
		}
		if req.ICMPType() != treq.icmpType {
			t.Error("Expected equal values: ", req.ICMPType(), treq.icmpType)
		}
		if req.ICMPCode() != treq.icmpCode {
			t.Error("Expected equal values: ", req.ICMPCode(), treq.icmpCode)
		}
		if req.Size() != treq.size {
			t.Error("Expected equal values: ", req.Size(), treq.size)
		}
		if req.TTL() != treq.ttl {
			t.Error("Expected equal values: ", req.TTL(), treq.ttl)
		}
		if req.TOS() != treq.tos {
			t.Error("Expected equal values: ", req.TOS(), treq.tos)
		}
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
		//Retrieves a nerPool instance, not the Singleton
		pool := newPool()

		//Golang code to get the mem stats of the runtime
		var mb, ma runtime.MemStats //The memStats variables to analyze memmory allocations
		runtime.GC()
		runtime.ReadMemStats(&mb) //Reading the before memory statistics

		//Instanciates and release a requester interface "counter" times
		var req Requester
		for i := 0; i < counter-1; i++ {
			req = pool.NewRequest() //Instantiates
			req.Release()           //Releases
		}

		//Now we will get one more instance of request
		req = pool.NewRequest()
		// Tells to release the request at the endo of the function
		defer req.Release()

		//Now we will run the garbage collector
		runtime.GC()
		//And red the memory statistics again
		runtime.ReadMemStats(&ma)

		if (ma.Mallocs - mb.Mallocs - 2) >= 5 {
			t.Error("Expected number of allocations be less than 5", (ma.Mallocs - mb.Mallocs - 2))
		}
		if req == nil {
			t.Error("Expected req to be not nil")
		}
		if req.Sequence() != (counter % 65536) {
			t.Error("Expected Sequence number to be ", (counter % 65536), ". got: ", req.Sequence())
		}
		//Testing Getting Default Values
		testGet(req, trequestMap["default"])

		//Testing Set Non Default Values
		testSet(req, trequestMap["nondefault"])

		//Testing Getting Non Default Values
		testGet(req, trequestMap["nondefault"])
	}
	return loopOverNewRequest
}
