package ggping

import "log"

const debug debugging = false // or flip to false

type debugging bool

func (d debugging) Printf(format string, args ...interface{}) {
	if d {
		log.Printf(format, args...)
	}
}
