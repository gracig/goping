package goping

import (
	"errors"
	"fmt"
	"net"
	"time"
)

var (
	ErrTakingTooLongTime = errors.New("Taking too long time while waiting for a response")
)

type request struct {
	messageImpl
	pool          Pooler
	target        Target
	ipproto       uint8
	ipversion     uint8
	timeout       time.Duration
	whenSent      time.Time
	responderChan chan<- Responder
}

func (r *request) ResponderChan() chan<- Responder {
	return r.responderChan
}
func (r *request) SetResponderChan(ch chan<- Responder) {
	r.responderChan = ch
}
func (r *request) Release() {
	r.responderChan = nil
	r.pool.ReleaseRequest(r)
}
func (r *request) SetTarget(t Target) {
	r.target = t
}
func (r *request) SetWhenSent(ws time.Time) {
	r.whenSent = ws
}
func (r *request) SetTimeout(timeout time.Duration) {
	r.timeout = timeout
}
func (r *request) SetIPVersion(ipversion uint8) {
	r.ipversion = ipversion
}
func (r *request) SetIPProto(proto uint8) {
	r.ipproto = proto
}
func (r *request) SetPool(p Pooler) {
	r.pool = p
}
func (r *request) Target() Target {
	return r.target
}
func (r *request) IPProto() uint8 {
	return r.ipproto
}
func (r *request) IPVersion() uint8 {
	return r.ipversion
}
func (r *request) Timeout() time.Duration {
	return r.timeout
}
func (r *request) WhenSent() time.Time {
	return r.whenSent
}

type response struct {
	messageImpl
	pool     Pooler
	request  Requester
	peer     net.IP
	err      error
	rtt      float64
	whenRecv time.Time
}

func (r *response) GoString() string {
	//TODO: Implement the GoString function
	return fmt.Sprintf("Implement String")
}
func (r *response) Release() {
	r.pool.ReleaseResponse(r)
}
func (m *response) SetPeer(v net.IP) {
	m.peer = v
}
func (r *response) SetRequest(req Requester) {
	r.request = req
}
func (r *response) SetWhenRecv(wr time.Time) {
	r.whenRecv = wr
}
func (r *response) SetRTT(rtt float64) {
	r.rtt = rtt
}
func (r *response) SetPool(p Pooler) {
	r.pool = p
}
func (r *response) SetError(err error) {
	r.err = err
}
func (r *response) Peer() net.IP {
	return r.peer
}
func (r *response) Request() Requester {
	return r.request
}
func (r *response) Error() error {
	return r.err
}
func (r *response) WhenRecv() time.Time {
	return r.whenRecv
}
func (r *response) RTT() float64 {
	return r.rtt
}

type messageImpl struct {
	sequence int
	icmpType byte
	icmpCode byte
	size     int
	ttl      int
	tos      int
}

func (m *messageImpl) SetTOS(v int) {
	m.tos = v
}
func (m *messageImpl) SetTTL(v int) {
	m.ttl = v
}
func (m *messageImpl) SetSize(v int) {
	m.size = v
}
func (m *messageImpl) SetICMPType(v byte) {
	m.icmpType = v
}
func (m *messageImpl) SetICMPCode(v byte) {
	m.icmpCode = v
}
func (m *messageImpl) SetSequence(s int) {
	m.sequence = s
}
func (m *messageImpl) Sequence() int {
	return m.sequence
}
func (m *messageImpl) ICMPType() byte {
	return m.icmpType
}
func (m *messageImpl) ICMPCode() byte {
	return m.icmpCode
}
func (m *messageImpl) Size() int {
	return m.size
}
func (m *messageImpl) TTL() int {
	return m.ttl
}
func (m *messageImpl) TOS() int {
	return m.tos
}
