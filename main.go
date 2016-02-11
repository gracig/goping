package main

import (
	"fmt"

	"github.com/gersongraciani/goping/protocol"
)

func main() {
	im := protocol.IcmpMessage{}
	im.Identifier = 2
	im.Sequence = 1
	im.Type = protocol.IcmpMessage_ECHO_REPLY
	im.Code = &protocol.IcmpMessage_SingleCode_{}
	fmt.Printf("Not Yet Functional! .ICMP Message  %#v\n", im)

}
