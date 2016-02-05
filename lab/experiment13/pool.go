package goping

import (
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	pool Pooler
)

func init() {
	pool = newPool()
}

func newPool() Pooler {
	return &poolImpl{
		reqpool: sync.Pool{
			New: func() interface{} {
				return new(request)
			},
		},
		respool: sync.Pool{
			New: func() interface{} {
				return new(response)
			},
		},
	}
}

/*
Pool returns the singleton Pooler of this package
*/
func Pool() Pooler {
	return pool
}

/*
Implements the Pooler Interface
*/
type poolImpl struct {
	reqpool sync.Pool
	respool sync.Pool
	seq     uint32
}

func (p *poolImpl) NewRequest() Requester {
	var r = p.reqpool.Get().(*request)
	r.SetTTL(60)
	r.SetTOS(0)
	r.SetSize(56)
	r.SetTarget(nil)
	r.SetICMPType(8)
	r.SetICMPCode(0)
	r.SetPool(p)
	r.SetIPProto(IPProtoICMP)
	r.SetIPVersion(IPV4)
	r.SetTimeout(3 * time.Second)
	r.SetWhenSent(time.Time{})
	r.sequence = int(atomic.AddUint32(&p.seq, 1) % (1 << 16))
	return r
}

func (p *poolImpl) NewResponse() Responder {
	var r = p.respool.Get().(*response)
	r.SetTTL(60)
	r.SetTOS(0)
	r.SetSize(56)
	r.SetRequest(nil)
	r.SetPeer(net.IPv4zero)
	r.SetICMPType(0)
	r.SetICMPCode(0)
	r.SetPool(p)
	r.SetWhenRecv(time.Time{})
	r.SetRTT(math.NaN())
	r.SetError(nil)
	return r
}

func (p *poolImpl) ReleaseRequest(r Requester) {
	r.SetPool(nil)
	p.reqpool.Put(r)
}

func (p *poolImpl) ReleaseResponse(r Responder) {
	r.SetPool(nil)
	p.respool.Put(r)
}
