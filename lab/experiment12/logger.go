package ggping

import "log"

const debug debugging = false  // or flip to false
const info debugging = true    // or flip to false
const severe debugging = true  // or flip to false
const ddebug debugging = false // or flip to false

type debugging bool

func (d debugging) Printf(format string, args ...interface{}) {
	if d {
		log.Printf(format, args...)
		//'		fmt.Printf(format, args...)
	}
}
func (d debugging) Println(args ...interface{}) {
	if d {
		log.Println(args...)
		//'		fmt.Printf(format, args...)
	}
}
