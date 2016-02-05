package goping

var (
	isRunning bool
)

type Request struct {
	To string
}

type Reply struct {
	Err error
	Rtt float64
}

type Summary struct {
	Replies []Reply
	Avg     float64
	Max     float64
	Min     float64
}
