package ggping

import (
	"fmt"
	"log"
	"math"
	"net"
	"runtime"
	"sort"
	"time"

	"golang.org/x/net/icmp"
)

//Local structs
type echoReply struct {
	when  time.Time
	bytes []byte
	size  int
	peer  net.Addr
}

const socketReadDeadLine int = 10

//Wakes Up Listener
func runListener(packetToMatch chan *echoReply) error {
	if isListening {
		debug.Printf("Warning: Trying to runListener but is Listening is true, ignoring command")
	} else {
		PacketConn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		if err != nil {
			return fmt.Errorf("Not possible to open raw socket ip4:icmp: %v", err)
		}

		go func() {
			//Make sure to cleanup resources after ReadDeadLine
			defer func() {
				PacketConn.Close()
				isListening = false
			}()
			rb := make([]byte, 1500) //The byte slice that will receive the icmp message
			for {
				if err := PacketConn.SetReadDeadline(time.Now().Add(time.Duration(socketReadDeadLine) * time.Second)); err != nil {
					log.Print("Could not Set the Read Dead Line to read the Icmp Socket:", err)
					return
				}
				size, peer, err := PacketConn.ReadFrom(rb) //Read from the socket
				if err != nil {
					log.Printf("Error reading packets from Socket: %v", err)
					return //Exit the loop in case of socket read error
				}
				packetToMatch <- &echoReply{when: time.Now(), bytes: rb, size: size, peer: peer}
			}
		}()
	}
	return nil
}

//This function returns the elapsed time between now and the last PingResponse request. (PingResponse.When)
func (t Pong) timeSinceLastResponse() time.Duration {
	lastResponse := t.Responses[len(t.Responses)-1]
	return time.Since(lastResponse.When)
}

//Start process the ping
//Pingers should be implemented in the pingers.go (Just for convetion) and they should implement the interface Pinger
//First tests were made with ping command, but the performance was too bad. Maybe because each ping spawn a unix child process of the ping program. But it was too slow.
func processPing(pong *Pong) {
	pong.Responses = append(pong.Responses, &PingResponse{})           //appends a new response object to pong
	var response *PingResponse = pong.Responses[len(pong.Responses)-1] //assigns a reference to the last created response to response

	//Get the pinger from os. As a empty struct, no memory will be allocated for now.
	var pinger Pinger
	switch os := runtime.GOOS; os {
	case "linux":
		pinger = GoPinger{}
	default:
		response.Error = fmt.Errorf("There is no Pinger associated with GOOS: %s", os)
		return
	}

	//Ping based on request parameters
	//Return the Rtt,Error and When in the response struct
	pinger.Ping(pong.Request, response, len(pong.Responses))

	return

}

//Process response and reeturn bool if all responses were received
func processPong(pong *Pong) bool {
	if len(pong.Responses) >= pong.Request.MaxPings {
		return true
	}
	return false
}

//This function is responsible to summarize all the responses from a Pong
func processSummary(pong *Pong) {

	pong.Summary = &PingSummary{}           //Creates a PingSummary object
	var summary *PingSummary = pong.Summary //References the pongSummary to summary

	//for loop that:
	//	Populate the validRtts array with the Rtt values
	//	Increments PingsSent and PingsReceived statistics
	//	PingsSent in incremented by each response either successful or not
	//	PingsReceived are only incremented by succesful responses (inside the response.Ok if)
	validRtts := make([]float64, 0, len(pong.Responses))
	for _, response := range pong.Responses {
		summary.PingsSent++
		if response.Error == nil {
			validRtts = append(validRtts, response.Rtt)
			summary.PingsReceived++
		}
	}

	if summary.PingsSent <= 0 {
		pong.Error = fmt.Errorf("No Pings were sent. Could not summarize")
		return
	}

	//Calculate PacketLoss
	summary.PacketLoss = (float64(summary.PingsSent) - float64(summary.PingsReceived)) / float64(summary.PingsSent) * 100

	//Calculate Packet Success
	summary.PacketSuccess = (float64(summary.PingsReceived) / float64(summary.PingsSent)) * 100

	//Return if all packets were lost. In this case there is no need to calculate the results
	if summary.PacketLoss == 100.0 {
		return
	}

	//Inner Function that Calculates RttAvg, RttMax and RttMin in a slice
	stats := func(values []float64) (min, max, avg float64) {
		var count, sum float64
		for i, v := range values {
			count++
			sum += float64(v)
			if i == 0 {
				min = float64(v)
				max = float64(v)
			} else {
				if float64(v) < min {
					min = float64(v)
				}
				if float64(v) > max {
					max = float64(v)
				}
			}
		}
		avg = sum / count
		return
	}

	//Assigns the calculated min, max and averago of the validRtts to the summary variables:
	//RttMin, RttMax and RttAvg respectively
	summary.RttMin, summary.RttMax, summary.RttAvg = stats(validRtts)

	//Find the xpercentil
	perc := pong.Request.Percentil //Retrieve the percentil attribute from the Request Object
	if perc <= 0 {
		return //If perc <=0 exit function, no percentil will be calculated at all
	}
	sort.Sort(sort.Float64Slice(validRtts))                                           //Sorting values to calculate the percentil
	index := int(math.Ceil(float64(len(validRtts)) * (float64(perc) / 100.0)))        //Finds the index of the percentil in the array
	summary.RttPerc = float64(validRtts[index-1])                                     //Assigns the percentil value found on index to RttPerc
	validPercRtts := validRtts[0:index]                                               //Reslice validRtts removing values above the found index
	summary.RttMinPerc, summary.RttMaxPerc, summary.RttAvgPerc = stats(validPercRtts) //Calculates Min,Max and Rtt on the new slice

	return
}
