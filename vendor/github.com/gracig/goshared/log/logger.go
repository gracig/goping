package log

import (
	"log"
	"os"
)

//define a type for log information
type logger bool

var (
	Debug, Info, Severe logger
)

func init() {
	var loglevel string = os.Getenv("GO_SHARED_LOG_LEVEL")
	switch loglevel {
	case "DEBUG":
		Debug, Info, Severe = true, true, true
	case "INFO":
		Debug, Info, Severe = false, true, true
	case "SEVERE":
		Debug, Info, Severe = false, false, true
	default:
		Debug, Info, Severe = false, true, true
	}
}

func (d logger) Printf(format string, args ...interface{}) {
	if d {
		log.Printf(format, args...)
		//'		fmt.Printf(format, args...)
	}
}
func (d logger) Println(args ...interface{}) {
	if d {
		log.Println(args...)
		//'		fmt.Printf(format, args...)
	}
}
