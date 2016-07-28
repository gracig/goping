package goping

import (
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewRequest(t *testing.T) {
	cfg := Config{Count: 1, Interval: time.Duration(1 * time.Second), PacketSize: 56, TOS: 16, TTL: 64, Timeout: (3 * time.Second)}
	g := New(cfg, &mockPinger{}, &mockSeqGen{seqmap: make(map[uint64]int)}, &mockIDGen{})

	for i := 0; i < 100; i++ {
		req := g.NewRequest("hostname", map[string]string{"id": "1"})
		if req.Config.Count != cfg.Count {
			t.Errorf("No match req.Config.Count. Expected: [%v], Got: [%v]", cfg.Count, req.Config.Count)
		}
		if req.Config.Interval != cfg.Interval {
			t.Errorf("No match req.Config.Interval. Expected: [%v], Got: [%v]", cfg.Interval, req.Config.Interval)
		}
		if req.Config.PacketSize != cfg.PacketSize {
			t.Errorf("No match req.Config.PacketSize. Expected: [%v], Got: [%v]", cfg.PacketSize, req.Config.PacketSize)
		}
		if req.Config.TOS != cfg.TOS {
			t.Errorf("No match req.Config.TOS. Expected: [%v], Got: [%v]", cfg.TOS, req.Config.TOS)
		}
		if req.Config.TTL != cfg.TTL {
			t.Errorf("No match req.Config.TTL. Expected: [%v], Got: [%v]", cfg.TTL, req.Config.TTL)
		}
		if req.Config.Timeout != cfg.Timeout {
			t.Errorf("No match req.Config.Timeout. Expected: [%v], Got: [%v]", cfg.Timeout, req.Config.Timeout)
		}
		id := uint64(i + 1)
		if req.ID != id {
			t.Errorf("No match req.ID. Expected: [%v], Got: [%v]", id, req.ID)
		}
		if req.UserData == nil {
			t.Errorf("req.UserData was expected to be initialized")
		}
		if req.UserData["id"] != "1" {
			t.Errorf("No Match UserData['id']. Expected: [%v], Got: [%v]", "1", req.UserData["id"])
		}
		if req.Host != "hostname" {
			t.Errorf("No match req.Host. Expected: [%v], Got: [%v]", "hostname", req.Host)
		}
		if req.Sent != 0 {
			t.Errorf("No match req.Sent. Expected: [%v], Got: [%v] ostname", 0, req.Sent)
		}
	}
}

type answer struct {
	err error
	raw RawResponse
}
type mockIDGen struct {
	status uint64
}

func (m *mockIDGen) Next() uint64 {
	return atomic.AddUint64(&(m.status), 1)
}

type mockSeqGen struct {
	seqmap map[uint64]int
}

func (m *mockSeqGen) Next(rid uint64) int {
	m.seqmap[rid]++
	return m.seqmap[rid] + int(rid)*1000
}

type mockPinger struct {
	answers map[int]answer
}

func (m *mockPinger) Start(pid int) (ping chan<- SeqRequest, pong <-chan RawResponse, donepong <-chan struct{}, err error) {
	in, out, doneIn, done := make(chan SeqRequest), make(chan RawResponse), make(chan struct{}, 1), make(chan struct{}, 1)
	ping, pong, donepong = in, out, done

	go func() {
		for {
			select {
			case recv, open := <-in:
				if open {
					go func() {
						<-time.After(time.Duration(m.answers[recv.Seq].raw.RTT))
						out <- m.answers[recv.Seq].raw
					}()
				} else {
					fmt.Println("Sending signal to doneIn")
					doneIn <- struct{}{}
					in = nil
				}
			case <-doneIn:
				fmt.Println("Pinger signaling that is done")
				done <- struct{}{}
				return
			}
		}
	}()

	return
}
func dur(i int) float64 {
	if i < 0 {
		return math.NaN()
	} else {
		return float64(time.Duration(i) * time.Millisecond)
	}
}

const (
	IPv4ECHOREPLY    = 1
	IPv4ECHOREQUEST  = 2
	IPv4TIMEEXCEEDED = 3
	IPv4UNREACHABLE  = 4
)

func msg(t int, seq int) []byte {
	switch t {
	case IPv4ECHOREQUEST:
		return []byte{8}
	case IPv4ECHOREPLY:
		return []byte{0}
	case IPv4TIMEEXCEEDED:
		return []byte{11}
	case IPv4UNREACHABLE:
		return []byte{3}
	default:
		return []byte{0}
	}
}

func TestGopinger(t *testing.T) {
	cfg := Config{Count: 10, Interval: time.Duration(500 * time.Millisecond), PacketSize: 56, TOS: 16, TTL: 64, Timeout: (500 * time.Millisecond)}

	pinger := &mockPinger{
		answers: map[int]answer{
			1001: {err: nil, raw: RawResponse{RTT: dur(110), ICMPMessage: msg(IPv4ECHOREPLY, 1001), Peer: net.ParseIP("192.168.0.1")}},
			1002: {err: nil, raw: RawResponse{RTT: dur(80), ICMPMessage: msg(IPv4ECHOREPLY, 1002), Peer: net.ParseIP("192.168.0.1")}},
			1003: {err: nil, raw: RawResponse{RTT: dur(65), ICMPMessage: msg(IPv4ECHOREPLY, 1003), Peer: net.ParseIP("192.168.0.1")}},
			1004: {err: nil, raw: RawResponse{RTT: dur(100), ICMPMessage: msg(IPv4ECHOREPLY, 1004), Peer: net.ParseIP("192.168.0.1")}},
			1005: {err: nil, raw: RawResponse{RTT: dur(99), ICMPMessage: msg(IPv4ECHOREPLY, 1005), Peer: net.ParseIP("192.168.0.1")}},
			1006: {err: nil, raw: RawResponse{RTT: dur(76), ICMPMessage: msg(IPv4ECHOREPLY, 1006), Peer: net.ParseIP("192.168.0.1")}},
			1007: {err: nil, raw: RawResponse{RTT: dur(80), ICMPMessage: msg(IPv4ECHOREPLY, 1007), Peer: net.ParseIP("192.168.0.1")}},
			1008: {err: nil, raw: RawResponse{RTT: dur(81), ICMPMessage: msg(IPv4ECHOREPLY, 1008), Peer: net.ParseIP("192.168.0.1")}},
			1009: {err: nil, raw: RawResponse{RTT: dur(150), ICMPMessage: msg(IPv4ECHOREPLY, 1009), Peer: net.ParseIP("192.168.0.1")}},
			1010: {err: nil, raw: RawResponse{RTT: dur(44), ICMPMessage: msg(IPv4ECHOREPLY, 1010), Peer: net.ParseIP("192.168.0.1")}},

			2001: {err: nil, raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 2001), Peer: net.ParseIP("192.168.0.2")}},
			2002: {err: nil, raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 2002), Peer: net.ParseIP("192.168.0.2")}},
			2003: {err: nil, raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 2003), Peer: net.ParseIP("192.168.0.2")}},
			2004: {err: nil, raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 2004), Peer: net.ParseIP("192.168.0.2")}},
			2005: {err: nil, raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 2005), Peer: net.ParseIP("192.168.0.2")}},
			2006: {err: nil, raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 2006), Peer: net.ParseIP("192.168.0.2")}},
			2007: {err: nil, raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 2007), Peer: net.ParseIP("192.168.0.2")}},
			2008: {err: nil, raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 2008), Peer: net.ParseIP("192.168.0.2")}},
			2009: {err: nil, raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 2009), Peer: net.ParseIP("192.168.0.2")}},
			2010: {err: nil, raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 2010), Peer: net.ParseIP("192.168.0.2")}},

			3001: {err: errors.New("Address Not Resolved"), raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 3001), Peer: net.ParseIP("192.168.0.3")}},
			3002: {err: errors.New("Address Not Resolved"), raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 3002), Peer: net.ParseIP("192.168.0.3")}},
			3003: {err: errors.New("Address Not Resolved"), raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 3003), Peer: net.ParseIP("192.168.0.3")}},
			3004: {err: errors.New("Address Not Resolved"), raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 3004), Peer: net.ParseIP("192.168.0.3")}},
			3005: {err: errors.New("Address Not Resolved"), raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 3005), Peer: net.ParseIP("192.168.0.3")}},
			3006: {err: errors.New("Address Not Resolved"), raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 3006), Peer: net.ParseIP("192.168.0.3")}},
			3007: {err: errors.New("Address Not Resolved"), raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 3007), Peer: net.ParseIP("192.168.0.3")}},
			3008: {err: errors.New("Address Not Resolved"), raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 3008), Peer: net.ParseIP("192.168.0.3")}},
			3009: {err: errors.New("Address Not Resolved"), raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 3009), Peer: net.ParseIP("192.168.0.3")}},
			3010: {err: errors.New("Address Not Resolved"), raw: RawResponse{RTT: dur(10000), ICMPMessage: msg(IPv4ECHOREPLY, 3010), Peer: net.ParseIP("192.168.0.3")}},
		},
	}

	//Create and populate a map of answers indexed by a sequence number
	chkMap := make(map[int]answer)
	for k, v := range pinger.answers {
		chkMap[k] = v
	}

	//Instantiate a new pinger
	g := New(cfg, pinger, &mockSeqGen{seqmap: make(map[uint64]int)}, &mockIDGen{})

	//Start the ping engine and get the in and out channels
	ping, pong, err := g.Start()
	if err != nil {
		t.Errorf("Error not expected")
	}

	//Start a goroutine to send all the requests to the ping channel
	go func() {
		for i := 0; i < 3; i++ {
			ping <- g.NewRequest("hostname"+strconv.Itoa(i+1), map[string]string{"i": strconv.Itoa(i + 1)})
		}
		close(ping) //<- This should signal to close pong
	}()

	//Start reading the pong channel
	for r := range pong {
		if a, ok := chkMap[r.Seq]; !ok {
			t.Errorf("Sequence was not found in pinger.answers [%v]", r.Seq)
		} else {
			delete(chkMap, r.Seq)
			fmt.Printf("Received response %v\n", r)

			//Check for timeouts
			if a.raw.RTT == dur(10000) && a.err == nil {
				if r.Err != ErrTimeout {
					t.Errorf("Error Expected: %v Got: %v", ErrTimeout, r.Err)
				}
			}

			if a.err != nil {
				if r.Err == nil {
					t.Errorf("Error Expected: %v Got: %v", a.err, r.Err)
				}
				if r.ICMPMessage != nil {
					t.Errorf("ICMPMessage expected to be nil. Got: %v", r.ICMPMessage)
				}
			}

			if r.Err != nil {
				if !math.IsNaN(r.RTT) {
					t.Errorf("RTT should be NaN on errors")
				}
			}
		}
	}
	if len(chkMap) > 0 {
		t.Errorf("There are remaining answers not consumed: %v", pinger.answers)
	}
}
