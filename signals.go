package goreact

import (
	"log"

	uuid "github.com/satori/go.uuid"
)

// "log"

type NextEvent func(v interface{})
type FailedEvent func(err error)
type CompletedEvent func(completed bool)

type NextEventWithMeta func(v interface{}, metas ...interface{})
type FailedEventWithMeta func(err error, metas ...interface{})
type CompletedEventWithMeta func(completed bool, metas ...interface{})

type Injector interface {
	SendFailed(err error)
	SendNext(value interface{})
	SendCompleted()
}

type Observer interface {
	GetId() uuid.UUID
	Injector
	// ListenNext(next NextEvent)
	// ListenFailed(failed FailedEvent)
	// ListenCompleted(completed CompletedEvent)
}

type DisposerList []Disposer

func (dl DisposerList) Dispose() {
	for _, d := range dl {
		d.Dispose()
	}
}

type Disposer struct {
	id          uuid.UUID
	unsubscribe chan<- uuid.UUID
}

func (d Disposer) Dispose() {
	if d.unsubscribe != nil {
		d.unsubscribe <- d.id
	}
}

func makeDisposer(sig *Signal) Disposer {
	return Disposer{id: uuid.NewV1(), unsubscribe: sig.unsubscribe}
}

type Signal struct {
	// isCompleted bool
	id          uint64
	injector    *ChanInjector
	observers   map[uuid.UUID]Observer
	subscribe   chan Observer
	unsubscribe chan uuid.UUID
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
		case obs, ok := <-s.subscribe:
			if ok == false {
				log.Println("BREAK---completed", s.id)
				break L
			}
			s.observers[obs.GetId()] = obs
		case idObs, ok := <-s.unsubscribe:

			if ok == false {
				log.Println("BREAK---completed", s.id)
				break L
			}
			delete(s.observers, idObs)
		}
	}
	log.Println("*******END OF RUN*********", s.id)
}

func (s *Signal) sendNext(v interface{}) {
	for _, obs := range s.observers {
		obs.SendNext(v)
	}
}

func (s *Signal) sendCompleted() {
	for _, obs := range s.observers {
		obs.SendCompleted()
	}
	s.observers = map[uuid.UUID]Observer{}
}

func (s *Signal) sendFailed(err error) {
	for _, obs := range s.observers {
		obs.SendFailed(err)
	}
}

func (s *Signal) ListenNext(next NextEvent) Disposer {
	// log.Println("add listen next :", next)
	disp := makeDisposer(s)
	cObs := Observe{next: next, id: disp.id}
	s.subscribe <- cObs
	return disp
}
func (s *Signal) ListenFailed(failed FailedEvent) Disposer {
	disp := makeDisposer(s)
	cObs := Observe{failed: failed, id: disp.id}
	s.subscribe <- cObs
	return disp

}
func (s *Signal) ListenCompleted(completed CompletedEvent) Disposer {
	disp := makeDisposer(s)
	cObs := Observe{completed: completed, id: disp.id}
	s.subscribe <- cObs
	return disp

}

func (s *Signal) Listen(next NextEvent, failed FailedEvent, completed CompletedEvent) Disposer {
	disp := makeDisposer(s)
	cObs := Observe{next: next, failed: failed, completed: completed, id: disp.id}
	s.subscribe <- cObs
	return disp
}

func (s *Signal) ListenNextWithMeta(next NextEventWithMeta, metas ...interface{}) Disposer {
	// log.Println("add listen next :", next)
	disp := makeDisposer(s)
	cObs := MetaObserve{next: next, id: disp.id, metas: metas}
	s.subscribe <- cObs
	return disp
}
func (s *Signal) ListenFailedWithMeta(failed FailedEventWithMeta, metas ...interface{}) Disposer {
	disp := makeDisposer(s)
	cObs := MetaObserve{failed: failed, id: disp.id, metas: metas}

	s.subscribe <- cObs
	return disp

}
func (s *Signal) ListenCompletedWithMeta(completed CompletedEventWithMeta, metas ...interface{}) Disposer {
	disp := makeDisposer(s)
	cObs := MetaObserve{completed: completed, id: disp.id, metas: metas}
	s.subscribe <- cObs
	return disp

}

func (s *Signal) ListenWithMeta(next NextEventWithMeta, failed FailedEventWithMeta, completed CompletedEventWithMeta, metas ...interface{}) Disposer {
	disp := makeDisposer(s)
	cObs := MetaObserve{next: next, failed: failed, completed: completed, id: disp.id, metas: metas}
	s.subscribe <- cObs
	return disp
}

func NewEmptySignal() *Signal {
	i := NewChanInjector()
	o := make(map[uuid.UUID]Observer, 0)
	chSub := make(chan Observer)
	chUnSub := make(chan uuid.UUID)
	s := &Signal{injector: i, observers: o, subscribe: chSub, unsubscribe: chUnSub}
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
	o := make(map[uuid.UUID]Observer, 0)
	chSub := make(chan Observer)
	chUnSub := make(chan uuid.UUID)
	s := &Signal{injector: i, observers: o, subscribe: chSub, unsubscribe: chUnSub}
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



//Operators
func (s *Signal) On(next NextEvent, failed FailedEvent, completed CompletedEvent) *Signal {

	if next != nil {
		s.ListenNext(func(nextValue interface{}) {
			next(nextValue)
		})
	}
	if failed != nil {
		s.ListenFailed(func(err error) {
			failed(err)
		})
	}
	if completed != nil {
		s.ListenCompleted(func(completedValue bool) {
			completed(completedValue)
		})
	}
	return s
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

func Merge(inners ...*Signal) *Signal {
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
	return pipe.Signal
}

func (s *Signal) Merge(inners ...*Signal) *Signal {
	all := append(inners, s)
	return Merge(all...)
}

func (s *Signal) Filter(filter func(value interface{}) (include bool)) *Signal {
	return NewSignal(func(obs Injector) {
		sig := s
		sig.ListenNext(func(next interface{}) {
			if filter(next) == true {
				obs.SendNext(next)
			}
		})
		sig.ListenFailed(func(err error) {
			obs.SendFailed(err)
		})
		sig.ListenCompleted(func(completed bool) {
			obs.SendCompleted()
		})
	})

}
