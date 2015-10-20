package ggping

import (
	"fmt"
	"testing"
)

func TestPing(t *testing.T) {

	fmt.Println("Pinging localhost")
	pong, err := Ping(&Request{To: "localhost", Timeout: 3})
	fmt.Printf("%v [%v]", pong, err)
}
