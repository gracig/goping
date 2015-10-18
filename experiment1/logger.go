package ggping

import "log"

const debug debugging = true // or flip to false

type debugging bool

func (d debugging) Printf(format string, args ...interface{}) {
	if d {
		log.Printf(format, args...)
	}
}
