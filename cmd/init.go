package main

import (
	"flag"
	"os"

	"github.com/gracig/goshared/log"
)

func init() {
	var help bool
	flag.BoolVar(&help, "help", false, "Display Help Message")
	flag.BoolVar(&help, "h", false, "Display Help Message")

	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}

	if log.DebugEnabled {
		log.Debug.Printf("Variable %v=%v\n", "help", help)
	}

}
