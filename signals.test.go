package goreact

import (
	"testing"
	"time"
)

func TestSignals(t *testing.T) {
	sig := NewSignal(func(obs Injector) {
		tick := time.NewTicker(200 * time.Millisecond)
		for idx, _ := range <-tick.C {
			if idx == 1000 {
				tick.Stop()
				break
			}
			obs.SendNext(idx)
		}
	})
	sig.ListenNext()

}
