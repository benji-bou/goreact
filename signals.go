package goreact

import (
	"log"
)

// "log"

type NextEvent func(v interface{})
type FailedEvent func(err error)
type CompletedEvent func(completed bool)

type Injector interface {
	SendFailed(err error)
	SendNext(value interface{})
	SendCompleted()
}

type Observer interface {
	Injector
	// ListenNext(next NextEvent)
	// ListenFailed(failed FailedEvent)
	// ListenCompleted(completed CompletedEvent)
}

var (
	idDisp uint64 = 0
	idSig  uint64 = 0
)

type DisposerList []Disposer

func (dl DisposerList) Dispose() {
	for _, d := range dl {
		d.Dispose()
	}
}

type Disposer struct {
	id     uint64
	signal *Signal
}

func (d Disposer) Dispose() {
	if d.signal != nil {
		delete(d.signal.Observers, d.id)
	}
}

func makeDisposer(sig *Signal) Disposer {
	idDisp++
	return Disposer{id: idDisp, signal: sig}
}

type Signal struct {
	// isCompleted bool
	id        uint64
	injector  *ChanInjector
	Observers map[uint64]Observer
}

func (s *Signal) On(next NextEvent, failed FailedEvent, completed CompletedEvent) *Signal {
	return NewSignal(func(obj Injector) {
		if next != nil {
			s.ListenNext(func(nextValue interface{}) {
				next(nextValue)
				obj.SendNext(nextValue)
			})
		}
		if failed != nil {
			s.ListenFailed(func(err error) {
				failed(err)
				obj.SendFailed(err)
			})
		}
		if completed != nil {
			s.ListenCompleted(func(completedValue bool) {
				completed(completedValue)
				obj.SendCompleted()
			})
		}
	})
}

func (s *Signal) OnNext(side NextEvent) *Signal {
	return s.On(side, nil, nil)
}

func (s *Signal) Map(transformer func(value interface{}) (interface{}, error)) *Signal {
	pipe := NewPipe()
	s.ListenNext(func(next interface{}) {
		newValue, err := transformer(next)
		if err != nil {
			pipe.Injector.SendFailed(err)
			return
		}
		pipe.Injector.SendNext(newValue)

	})
	s.ListenFailed(func(err error) {
		pipe.Injector.SendFailed(err)
	})
	s.ListenCompleted(func(completed bool) {
		pipe.Injector.SendCompleted()
	})
	return pipe.Signal
}

func (s *Signal) Merge(inners ...*Signal) *Signal {
	pipe := NewPipe()
	for _, inner := range inners {
		inner.Listen(func(event interface{}) {
			pipe.Injector.SendNext(event)
		}, func(err error) {
			pipe.Injector.SendFailed(err)
		}, func(completed bool) {
			pipe.Injector.SendCompleted()
		})
	}
	s.Listen(func(event interface{}) {
		pipe.Injector.SendNext(event)
	}, func(err error) {
		pipe.Injector.SendFailed(err)
	}, func(completed bool) {
		pipe.Injector.SendCompleted()
	})
	return pipe.Signal
}

func (s *Signal) run() {
L:
	for {
		select {
		case err, ok := <-s.injector.failed:
			if ok == false {
				log.Println("BREAK---failed", s.id)
				break L
			}
			s.sendFailed(err)
		case next, ok := <-s.injector.next:
			if ok == false {
				log.Println("BREAK---next", s.id)
				break L
			}
			s.sendNext(next)
		case _, ok := <-s.injector.completed:
			if ok == false {
				log.Println("BREAK---completed", s.id)
				break L
			}
			s.sendCompleted()
			break L
		}
	}
	log.Println("*******END OF RUN*********", s.id)
}

func (s *Signal) sendNext(v interface{}) {

	for _, obs := range s.Observers {
		obs.SendNext(v)
	}
}

func (s *Signal) sendCompleted() {
	for _, obs := range s.Observers {
		obs.SendCompleted()
	}
	s.Observers = map[uint64]Observer{}
	s.injector.Close()
}

func (s *Signal) sendFailed(err error) {
	for _, obs := range s.Observers {
		obs.SendFailed(err)
	}
}

func (s *Signal) ListenNext(next NextEvent) Disposer {
	// log.Println("add listen next :", next)
	cObs := Observe{next: next}
	disp := makeDisposer(s)
	s.Observers[disp.id] = cObs
	return disp
}
func (s *Signal) ListenFailed(failed FailedEvent) Disposer {
	cObs := Observe{failed: failed}
	disp := makeDisposer(s)
	s.Observers[disp.id] = cObs
	return disp

}
func (s *Signal) ListenCompleted(completed CompletedEvent) Disposer {
	cObs := Observe{completed: completed}
	disp := makeDisposer(s)
	s.Observers[disp.id] = cObs
	return disp

}

func (s *Signal) Listen(next NextEvent, failed FailedEvent, completed CompletedEvent) Disposer {
	cObs := Observe{next: next, failed: failed, completed: completed}
	disp := makeDisposer(s)
	s.Observers[disp.id] = cObs
	return disp

}

func NewEmptySignal() *Signal {
	i := NewChanInjector()
	o := make(map[uint64]Observer, 0)
	s := &Signal{injector: i, Observers: o}
	s.id = idSig
	idSig++
	go s.run()
	return s
}

func NewSignal(generator func(obs Injector)) *Signal {
	s := NewEmptySignal()
	go generator(s.injector)
	return s
}

type Pipe struct {
	Signal   *Signal
	Injector Injector
}

func NewPipe() Pipe {
	i := NewChanInjector()
	o := make(map[uint64]Observer, 0)
	s := &Signal{injector: i, Observers: o}
	s.id = idSig
	idSig++
	go s.run()
	return Pipe{Signal: s, Injector: i}
}

type ChanInjector struct {
	isAvailable bool
	failed      chan error
	next        chan interface{}
	completed   chan bool
}

func NewChanInjector() *ChanInjector {
	errc := make(chan error)
	nextc := make(chan interface{})
	completedc := make(chan bool)
	return &ChanInjector{isAvailable: true, failed: errc, next: nextc, completed: completedc}
}

func (i *ChanInjector) Close() {
	i.isAvailable = false
	close(i.failed)
	close(i.next)
	close(i.completed)
}

func (i *ChanInjector) SendFailed(err error) {
	if i.isAvailable == true {
		i.failed <- err
	}
}

func (i *ChanInjector) SendNext(value interface{}) {
	if i.isAvailable == true {
		i.next <- value
	}
}

func (i *ChanInjector) SendCompleted() {
	if i.isAvailable == true {
		i.completed <- true
	}
}

type Observe struct {
	next      NextEvent
	failed    FailedEvent
	completed CompletedEvent
}

func (o Observe) SendFailed(err error) {
	if o.failed != nil {
		o.failed(err)
	}
}

func (o Observe) SendNext(value interface{}) {
	if o.next != nil {
		o.next(value)
	}

}

func (o Observe) SendCompleted() {
	if o.completed != nil {
		o.completed(true)
	}

}
