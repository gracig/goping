package goping

import (
	"errors"
	"sync"
)

/*** This are interfaces and utility functions to work with ping extensions***/

//Pinger is responsible for  the low implementation to send and receive pings over the network
type Pinger interface {
	//Start initiate the channels where the pinger will receive requests and should send the responses.
	Start(pid int) (ping chan<- SeqRequest, pong <-chan RawResponse, done <-chan struct{}, err error)
}

/*** Errors ***/
var (
	ErrTimeout             = errors.New("Timeout")
	ErrDstUnreachable      = errors.New("Destination Unreachable")
	ErrParamProblem        = errors.New("Parameter problem")
	ErrTimeExceeded        = errors.New("Time Exceeded")
	ErrRedirect            = errors.New("Redirect Message")
	ErrUnknown             = errors.New("Unknown Packet")
	ErrPingerNotRegistered = errors.New("Ping not registered")

	ErrCouldNotStartPinger = errors.New("Could not start pinger")
)

/*** Pinger Registry ***/
var (
	pregistry      map[string]Pinger
	pregistryMutex sync.Mutex
)

//RegPingerAdd Used to register new low implementations of the Pinger interface
func RegPingerAdd(name string, pinger Pinger) {
	pregistryMutex.Lock()
	defer pregistryMutex.Unlock()
	if pregistry == nil {
		pregistry = make(map[string]Pinger)
	}
	pregistry[name] = pinger
}

//RegPingerGet returns a Pinger implementation
func RegPingerGet(name string) (Pinger, error) {
	pregistryMutex.Lock()
	defer pregistryMutex.Unlock()
	if pinger, ok := pregistry[name]; ok {
		return pinger, nil
	}
	return nil, ErrPingerNotRegistered
}

//RegPingerList returns a list of Pinger Implementations that exists in runtime
func RegPingerList() []string {
	pregistryMutex.Lock()
	defer pregistryMutex.Unlock()
	names := make([]string, len(pregistry))
	i := 0
	for name := range pregistry {
		names[i] = name
		i++
	}
	return names
}
