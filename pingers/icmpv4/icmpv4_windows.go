package icmpv4

import (
	"fmt"
	"syscall"

	"github.com/gracig/goping"
)

func New() goping.Pinger {
	return &pinger{}
}

//Pinger is the type the implements goping.Pinger interface
type pinger struct {
}

//Start is the implementation of the method goping.Pinger.Start
func (p pinger) Start(pid int) (ping chan<- goping.SeqRequest, pong <-chan goping.RawResponse, done <-chan struct{}, err error) {
	var icmp syscall.Handle
	icmp, err = syscall.LoadLibrary("ICMP.DLL")
	//Loading DLL
	if err != nil {
		err = fmt.Errorf("LoadLibrary() failed: Unable to locate ICMP.DLL: %v", err)
		return
	}
	//Creating File
	_, err = syscall.GetProcAddress(icmp, "IcmpCreateFile")
	if err != nil {
		err = fmt.Errorf("Could not get ProcFile IcmpCreateFile: %v", err)
		return
	}
	err = fmt.Errorf("Non implemetation in Windows %v", err)
	return
}
