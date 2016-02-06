package goping

import (
	"errors"
	"math"
	"runtime"
	"sync"
	"time"
)

var ctl Controller

const (
	GET = iota
	PUT
	DEL
)

var (
	ErrNoRegisteredOSPinger = errors.New("No registered OSPinger")
	ErrNoSuchOperation      = errors.New("No such operation. only GET PUT or DEL are accepted")
	ErrSeqSlotIsOccupied    = errors.New("Error while trying to use Sequence slot to wait for a response")
	ErrResponseTimeout      = errors.New("Response timeout")
	ErrNoResponderChannel   = errors.New("Request has no Responder Channel to receive a reply")
)

func init() {
	ctl = newController()
	go ctl.(*controllerImpl).checkPongLoop()
	go ctl.(*controllerImpl).checkTimeoutLoop()
}

//Return the Ping Controller of this package
func Ctl() Controller {
	return ctl
}

//To  change the builtin implementation of PingController.
func SetCtl(c Controller) {
	ctl = c
}

func newController() Controller {
	return &controllerImpl{
		//The OSPinger Map
		ospMap: make(map[[10]byte]OSPinger),

		//The channels for checkTimeoutLoop
		chTimeout:     make(chan *seqtimeout, math.MaxUint16),
		chTimeoutDone: make(chan struct{}, 1),
		seqtimePool:   sync.Pool{New: func() interface{} { return new(seqtimeout) }},

		//The channels for checkPongLoop
		chPing:     make(chan Requester),
		chPong:     make(chan Responder, math.MaxUint16),
		chPongT:    make(chan int, math.MaxUint16),
		chPongDone: make(chan struct{}, 1),
	}
}

type seqtimeout struct {
	seq     int
	when    time.Time
	timeout time.Duration
}

//Builtin implementation of Controller
type controllerImpl struct {
	ospMap map[[10]byte]OSPinger

	chTimeout     chan *seqtimeout
	chTimeoutDone chan struct{}
	seqtimePool   sync.Pool

	chPing     chan Requester
	chPong     chan Responder
	chPongT    chan int
	chPongDone chan struct{}
}

func (p *controllerImpl) checkPongLoop() {
	var requestArray [math.MaxUint16]Requester
	for {
		select {

		//Receives new requests
		case req := <-p.chPing:

			//Verifies if responseArray is free for this Sequence
			if requestArray[req.Sequence()] != nil {
				panic(ErrSeqSlotIsOccupied)
			}
			//Attach the request object to the request Array using the sequence as index
			requestArray[req.Sequence()] = req

		//Receives responses from OSPinger
		case resp := <-p.chPong:

			req := requestArray[resp.Sequence()]
			if req == nil {
				//Request no longer found, could have been timedout. Releasing Response
				resp.Release()
				continue
			}
			//Removing the request from the array
			requestArray[resp.Sequence()] = nil

			//Attaching the request to the response
			resp.SetRequest(req)

			//Send the response through the Responder Channel
			req.ResponderChan() <- resp

		//Receives Timeout messages from checkTimeoutLoop
		case seq := <-p.chPongT:
			req := requestArray[seq]
			if req == nil {
				//Request no longer found, could have been already replied. Ignoring
				continue
			}
			//Removing the request from the array
			requestArray[seq] = nil

			//Creating a new Response and setting the ErrResponseTimeout error
			resp := Pool().NewResponse()
			resp.SetSequence(seq)
			resp.SetRequest(req)
			resp.SetError(ErrResponseTimeout)

			//Sending the response through the Responder channel
			req.ResponderChan() <- resp

		case <-p.chPongDone:
			return
		}
	}
}

func (p *controllerImpl) checkTimeoutLoop() {
	for {
		select {

		case st := <-p.chTimeout:
			time.Sleep(st.timeout - time.Now().Sub(st.when))
			p.chPongT <- st.seq
			p.seqtimePool.Put(st)

		case <-p.chTimeoutDone:
			return
		}
	}
}

//Controller Methods
func (p *controllerImpl) Ping(r Requester) error {

	//Verifies if ResponderChannel is not nil
	if r.ResponderChan() == nil {
		return ErrNoResponderChannel
	}

	//Tries to retrieve the ospinger. Return error if not found
	osp, err := p.OSPinger(r.IPVersion(), r.IPProto(), runtime.GOOS)
	if err != nil {
		return err
	}

	//Send the request to the CheckPongLoop.
	p.chPing <- r

	//Call ospinger Ping method and store the WhenSent value
	r.SetWhenSent(osp.Ping(r))

	//Send the request to the CheckTimeoutLoop . It may be a Buffered Channel. To avoid the wait time
	st := p.seqtimePool.Get().(*seqtimeout)
	st.seq = r.Sequence()
	st.when = r.WhenSent()
	st.timeout = r.Timeout()
	p.chTimeout <- st

	return nil

}

func (p *controllerImpl) Pong(r Responder) error {

	//Send the response to the chPong Channel
	p.chPong <- r

	return nil
}

func (p *controllerImpl) RegisterOSPinger(ver uint8, prot uint8, goos string, osPinger OSPinger) (OSPinger, error) {
	return p.ospingerOps(PUT, ver, prot, goos, osPinger)
}
func (p *controllerImpl) UnRegisterOSPinger(ver uint8, prot uint8, goos string) (OSPinger, error) {
	return p.ospingerOps(DEL, ver, prot, goos, nil)
}
func (p *controllerImpl) OSPinger(ver uint8, prot uint8, goos string) (OSPinger, error) {
	return p.ospingerOps(GET, ver, prot, goos, nil)
}

func (p *controllerImpl) ospingerOps(op byte, ver uint8, prot uint8, goos string, osPinger OSPinger) (OSPinger, error) {

	//Build the Hash Key
	var key [10]byte
	copy(key[:2], []byte{ver, prot})
	copy(key[2:], goos)

	//Retrieve the actual OSPinger, if any
	var err error
	osp, ok := p.ospMap[key]

	//Do the desired operation
	switch op {

	case GET:
		//Get OSPinger from map
		if !ok {
			err = ErrNoRegisteredOSPinger
		}
	case PUT:
		//Put OSPinger on map
		p.ospMap[key] = osPinger
		osp = osPinger
	case DEL:
		//Delete OSPinger on map
		if ok {
			delete(p.ospMap, key)
		}
	default:
		err = ErrNoSuchOperation
	}
	return osp, err
}
