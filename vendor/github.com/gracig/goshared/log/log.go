package log

import (
	"io/ioutil"
	"log"
	"os"
)

const (
	debugPrefix  = "DEBUG:  "
	infoPrefix   = "INFO:   "
	warnPrefix   = "WARN:   "
	severePrefix = "SEVERE: "
	flags        = log.Ldate | log.Ltime | log.Lshortfile
)

var (
	//The loggers  that will be used
	Debug, Info, Warn, Severe *log.Logger

	//Flag to indicate is a log should be used
	DebugEnabled, InfoEnabled, WarnEnabled, SevereEnabled bool
)

func init() {
	//Read log level from Env Var
	var loglevel string = os.Getenv("GO_SHARED_LOG_LEVEL")

	//toogle cases
	switch loglevel {
	case "DEBUG":
		DebugEnabled, InfoEnabled, WarnEnabled, SevereEnabled = true, true, true, true
	case "INFO":
		DebugEnabled, InfoEnabled, WarnEnabled, SevereEnabled = false, true, true, true
	case "WARNING":
		DebugEnabled, InfoEnabled, WarnEnabled, SevereEnabled = false, false, true, true
	case "SEVERE":
		DebugEnabled, InfoEnabled, WarnEnabled, SevereEnabled = false, false, false, true
	default:
		DebugEnabled, InfoEnabled, WarnEnabled, SevereEnabled = false, true, true, true
	}

	//Initialize loggers
	if DebugEnabled {
		Debug = log.New(os.Stderr, debugPrefix, flags)
	} else {
		Debug = log.New(ioutil.Discard, debugPrefix, flags)
	}

	if InfoEnabled {
		Info = log.New(os.Stderr, infoPrefix, flags)
	} else {
		Info = log.New(ioutil.Discard, infoPrefix, flags)
	}

	if WarnEnabled {
		Warn = log.New(os.Stderr, warnPrefix, flags)
	} else {
		Warn = log.New(ioutil.Discard, warnPrefix, flags)
	}

	if SevereEnabled {
		Severe = log.New(os.Stderr, severePrefix, flags)
	} else {
		Severe = log.New(ioutil.Discard, severePrefix, flags)
	}
}
