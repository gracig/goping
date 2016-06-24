package main

const (
	ICMP_MINLEN = 28
	DEFDATALEN  = 64 - 8 //default data length
	MAXIPLEN    = 60
	MAXICMPLEN  = 76
	MAXPACKet   = (65536 - 60 - 8) //max packet size
	MAXWAIT     = 10               //max seconds to wait for a response
	NROUTES     = 9                //Number of record route slots
)
