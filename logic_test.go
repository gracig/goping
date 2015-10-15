package ggping

import "testing"

func TestProcessSummary(t *testing.T) {
	for i, tableTask := range testTable {
		//Create a new task: testTask with the request and responses from the testTable
		testTask := PingTask{Request: tableTask.Request, Responses: tableTask.Responses}
		//We will analyze the summary results after call ProcessSummary on the new created task
		ProcessSummary(&testTask)

		if testTask.Summary.PingsSent != tableTask.Summary.PingsSent {
			t.Error(i, "PingsSent Expected", tableTask.Summary.PingsSent, "got", testTask.Summary.PingsSent)
		}
		if testTask.Summary.PingsReceived != tableTask.Summary.PingsReceived {
			t.Error(i, "PingsReceived Expected", tableTask.Summary.PingsReceived, "got", testTask.Summary.PingsReceived)
		}
		if testTask.Summary.PacketLoss != tableTask.Summary.PacketLoss {
			t.Error(i, "PacketLoss Expected", tableTask.Summary.PacketLoss, "got", testTask.Summary.PacketLoss)
		}
		if testTask.Summary.PacketSuccess != tableTask.Summary.PacketSuccess {
			t.Error(i, "PacketSuccess Expected", tableTask.Summary.PacketSuccess, "got", testTask.Summary.PacketSuccess)
		}
		if testTask.Summary.RttAvg != tableTask.Summary.RttAvg {
			t.Error(i, "RttAvg Expected", tableTask.Summary.RttAvg, "got", testTask.Summary.RttAvg)
		}
		if testTask.Summary.RttMax != tableTask.Summary.RttMax {
			t.Error(i, "RttMax Expected", tableTask.Summary.RttMax, "got", testTask.Summary.RttMax)
		}
		if testTask.Summary.RttMin != tableTask.Summary.RttMin {
			t.Error(i, "RttMin Expected", tableTask.Summary.RttMin, "got", testTask.Summary.RttMin)
		}
		if testTask.Summary.RttPerc != tableTask.Summary.RttPerc {
			t.Error(i, "RttPerc Expected", tableTask.Summary.RttPerc, "got", testTask.Summary.RttPerc)
		}
		if testTask.Summary.RttAvgPerc != tableTask.Summary.RttAvgPerc {
			t.Error(i, "RttAvgPerc Expected", tableTask.Summary.RttAvgPerc, "got", testTask.Summary.RttAvgPerc)
		}
		if testTask.Summary.RttMaxPerc != tableTask.Summary.RttMaxPerc {
			t.Error(i, "RttMaxPerc Expected", tableTask.Summary.RttMaxPerc, "got", testTask.Summary.RttMaxPerc)
		}
		if testTask.Summary.RttMinPerc != tableTask.Summary.RttMinPerc {
			t.Error(i, "RttMinPerc Expected", tableTask.Summary.RttMinPerc, "got", testTask.Summary.RttMinPerc)
		}
	}
}

//Builds a test table to test various scenarios
//All the logic functions depends on the PingTask Object
//Our test table will be an array of PingTask objects,
//Various aspects of the logic will be tested using those
var testTable = []PingTask{
	{
		Request: PingRequest{
			HostDest:  "www.google.com",
			Timeout:   3,
			MaxPings:  10,
			MinWait:   2,
			Percentil: 90,
			UserMap: map[string]string{
				"device": "SGRC_1001",
				"key":    "value",
			},
		},
		Responses: []PingResponse{
			{Rtt: 80, Error: nil},
			{Rtt: 20, Error: nil},
			{Rtt: 103, Error: nil},
			{Rtt: 340, Error: nil},
			{Rtt: 60, Error: nil},
			{Rtt: 58, Error: nil},
			{Rtt: 89, Error: nil},
			{Rtt: 30, Error: nil},
			{Rtt: 10, Error: nil},
			{Rtt: 15, Error: nil},
		},
		Summary: PingSummary{
			PingsSent:     int(10),
			PingsReceived: int(10),
			PacketLoss:    float64(0),
			PacketSuccess: float64(100),
			RttAvg:        (80.0 + 20.0 + 103.0 + 60.0 + 58.0 + 89.0 + 30.0 + 10.0 + 15.0 + 340.0) / 10,
			RttMax:        float64(340),
			RttMin:        float64(10),
			RttPerc:       float64(103),
			RttAvgPerc:    (80.0 + 20.0 + 103.0 + 60.0 + 58.0 + 89.0 + 30.0 + 10.0 + 15.0) / 9,
			RttMaxPerc:    float64(103),
			RttMinPerc:    float64(10),
		},
		Error: nil,
	},
}
