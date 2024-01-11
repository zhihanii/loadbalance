package loadbalance

import "sync/atomic"

type round struct {
	state uint64
}

func (r *round) Next() uint64 {
	return atomic.AddUint64(&r.state, 1)
}

func newRound() *round {
	return &round{}
}
