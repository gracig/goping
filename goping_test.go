package goping

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

type mockLogger struct {
	t   *testing.T
	dbg bool
}

func (l mockLogger) Warn(fmt string, v ...interface{}) {
	l.t.Logf(fmt, v...)
}
func (l mockLogger) Info(fmt string, v ...interface{}) {
	l.t.Logf(fmt, v...)
}
func (l mockLogger) Severe(fmt string, v ...interface{}) {
	l.t.Logf(fmt, v...)
}
func (l mockLogger) IsDebug() bool {
	return l.dbg
}
func (l mockLogger) Debug(fmt string, v ...interface{}) {
	l.t.Logf(fmt, v...)
}

func TestNewRequest(t *testing.T) {
	cfg := Config{Count: 1, Interval: time.Duration(1 * time.Second), PacketSize: 56, TOS: 16, TTL: 64, Timeout: (3 * time.Second)}
	g := New(cfg, &mockLogger{}, &mockPinger{})

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
		if req.Id != id {
			t.Errorf("No match req.Id. Expected: [%v], Got: [%v]", id, req.Id)
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

type mockPinger struct {
	seqmap  map[uint64]int
	answers map[uint64]RawResponse
}

func (m *mockPinger) Ping(r Request) (future <-chan RawResponse, seq int, err error) {
	m.seqmap[r.Id]++
	seq = m.seqmap[r.Id] + int(r.Id)*1000
	future = make(chan RawResponse, 1)

	return
}

func TestGopinger(t *testing.T) {
	cfg := Config{Count: 10, Interval: time.Duration(500 * time.Millisecond), PacketSize: 56, TOS: 16, TTL: 64, Timeout: (3000 * time.Millisecond)}
	g := New(cfg, mockLogger{}, &mockPinger{
		seqmap: make(map[uint64]int),
	})
	ping, pong := g.Start()

	go func() {
		for i := 0; i < 10; i++ {
			ping <- g.NewRequest("hostname", map[string]string{"i": strconv.Itoa(i + 1)})
		}
		close(ping) //<- This should signal to close pong
	}()

	for resp := range pong {
		fmt.Printf("Reading a response: reqid: %v seq: %v err: %v RTT: %v\n", resp.Request.Id, resp.Seq, resp.Err, resp.RTT)
	}
}
