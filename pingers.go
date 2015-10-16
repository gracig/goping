package ggping

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type GoPinger struct{}     //
const icmpProtocol int = 1 //Used in the ParseMessage Functions
//Start listening for icmp packets. Should be root user.
//One socket will be opened for each ping worker. This way we can parallelizing the work.
//But every socket will receive every replies
//We have to search for the right packet (Id:request.Number seq: seq) before returning that to a channel
//And we have to start a go routine to quickly read the overhead packets and find the right one
func (p GoPinger) Ping(request *PingRequest, response *PingResponse, seq int) {

	//Tag the time on response before the ping command, to avoid wait more time
	response.When = time.Now()

	var dst *net.IPAddr = request.IP

	//Calling function to open Socket Connection on raw ip4:icmp
	PacketConn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		response.Error = fmt.Errorf("Not possible to open raw socket ip4:icmp: %v", err)
		return
	}

	//Make sure we close the socket. This will generate an error on the go function that reads the socket.
	//The error is because the function is trying to read from a socket that was closed.
	//I use that error to close the go function. Dont know if this is a right pattern, but logically this works
	defer PacketConn.Close()

	icmpReceived := make(chan bool) //Channel that this function use to indicate a ping has been received
	//This go function start reading all packets from the Socket in a infinite loop. When socket is closed this infinit loop is also closed too
	go func() {

		rb := make([]byte, 1500) //The byte slice that will receive the icmp message
		for {
			sizeInBytes, peer, err := PacketConn.ReadFrom(rb) //Read from the socket
			rt := time.Since(response.When)                   //Set the time the message arrived, will be used in Rtt in case of match

			if err != nil {
				debug.Printf("Connection Closed for Id: %v, Seq: %v", request.Number, seq)
				return //Exit the loop in case of socket read error
			}
			message, err := icmp.ParseMessage(icmpProtocol, rb[:sizeInBytes]) //Uses google code to parse the icmp Message
			if err != nil {
				debug.Printf("Error Parsing Message: %v", err)
			}
			if message.Type == ipv4.ICMPTypeEchoReply { //Message should be of type ICMPTypeEchoReply
				var body *icmp.Echo = message.Body.(*icmp.Echo)   //Type Assertion for icmp.Echo
				if body.ID == request.Number && body.Seq == seq { //Comparing Id and Seq from request with received message
					//Found Reply, marking rtt type and returning
					response.Rtt = rt.Seconds() * float64(time.Second/time.Millisecond)
					debug.Printf("ECHO REPLY FROM IP: %v BYTES: %v Id: %v, Seq: %v, RTT: %v", peer, sizeInBytes, request.Number, seq, response.Rtt)
					icmpReceived <- true //Communicates that the icmp was received before return
					return
				}
			}
		}
	}()

	debug.Printf("Host Destination Ip: %v\n", request.IP)

	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: request.Number, Seq: seq,
			Data: []byte("HELLO-R-U-THERE####All ALL OTHER STUFF!"),
		},
	}

	//Google code to marshal the header in a binary format to be sent over the socket
	wb, err := wm.Marshal(nil)
	if err != nil {
		response.Error = err
		return
	}

	//Send the packet over the socket connection. We hope to get a reply in the go routine in the beginning of this function
	//Otherwise we will receive a timeout and exit the function, closing the socket connection and the go routine subsequentially
	if _, err := PacketConn.WriteTo(wb, dst); err != nil {
		response.Error = err
		return
	}

	//Tag the time on response before the ping command, to avoid wait more time
	response.When = time.Now()

	select {
	case <-icmpReceived: //Received icmp response
		return
	case <-time.After(time.Second * time.Duration(request.Timeout)): //Icmp Response Timed out
		response.Error = fmt.Errorf("Ping Reply Timed  out")
		response.Rtt = 0
		return
	}

	return

}

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
