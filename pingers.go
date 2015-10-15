package ggping

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

//Pinger implementations goes here
//Defines the linux pinger. this should implement a ping using the SO ping command
//It should implement the Pinger interface to be called by the main ggping
type LinuxPinger struct{}

func (p LinuxPinger) Ping(request *PingRequest, response *PingResponse, seq int) {

	//Tag the time on response before the ping command, to avoid wait more time
	response.When = time.Now()

	//return //avoid processing

	//Run the command
	cmd := exec.Command(
		"ping",
		"-c 1",
		"-W",
		fmt.Sprintf("%v", request.Timeout),
		"-s",
		fmt.Sprintf("%v", request.Payload),
		"-Q",
		fmt.Sprintf("%v", request.Tos),
		request.HostDest,
	)
	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput
	err := cmd.Run()

	//If status code is different of 1 then
	if err != nil {
		response.Error = err
		response.Rtt = 0
	}

	//Getting stdout as a string reader of the previous output

	//Reading the output
	//Linux Response: rtt min/avg/max/mdev = 22.600/22.600/22.600/0.000 ms
	regex := regexp.MustCompile(`^rtt min/avg/max/mdev = (.*)/(.*)/(.*)/(.*) ms$`)
	scanner := bufio.NewScanner(cmdOutput)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		text := scanner.Text()

		if regex.MatchString(text) {
			matches := regex.FindStringSubmatch(text)
			response.Rtt, _ = strconv.ParseFloat(matches[2], 64)
			debug.Printf("Ping Response: %v\n", text)
			debug.Printf("Ping Capture:  Bytes: %v Rtt: %v\n", request.Payload, response.Rtt)
		} else {
			debug.Printf("Ping Unmatched Response: %v\n", text)
		}
	}

	return
}

type GoPinger struct{}

func (p GoPinger) Ping(request *PingRequest, response *PingResponse, seq int) {

	dst := request.IP
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		response.Error = fmt.Errorf("Not possible to open raw socket ip4:icmp. Maybe need root privilegesi: %v", err)
		return
	}
	defer c.Close()

	debug.Printf("Host Destination Ip: %v\n", request.IP)

	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: int(request.Number), Seq: 1 << uint(seq),
			Data: []byte("HELLO-R-U-THERE####All ALL OTHER STUFF!"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		response.Error = err
		return
	}

	rb := make([]byte, 1500)
	if err := c.SetReadDeadline(time.Now().Add(time.Duration(request.Timeout * float64(time.Second)))); err != nil {
		response.Error = err
		return
	}
	//Tag the time on response before the ping command, to avoid wait more time
	response.When = time.Now()
	if n, err := c.WriteTo(wb, dst); err != nil {
		response.Error = err
		return
	} else if n != len(wb) {
		response.Error = fmt.Errorf("got %v; want %v", n, len(wb))
		return
	}
	n, peer, err := c.ReadFrom(rb)
	response.Rtt = time.Since(response.When).Seconds() * float64(time.Second/time.Millisecond)
	if err != nil {
		response.Error = err
		response.Rtt = 0
		return
	} else {
	}
	debug.Printf("n: %v peer: %v err: %v", n, peer, err)
	return

}
