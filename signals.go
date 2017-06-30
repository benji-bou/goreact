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

type Signal struct {
	// isCompleted bool
	id        int
	injector  *ChanInjector
	Observers []Observer
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
			s.SendFailed(err)
		case next, ok := <-s.injector.next:
			if ok == false {
				log.Println("BREAK---next", s.id)
				break L
			}
			s.SendNext(next)
		case _, ok := <-s.injector.completed:
			if ok == false {
				log.Println("BREAK---completed", s.id)
				break L
			}
			s.SendCompleted()
			break L
		}
	}
	log.Println("*******END OF RUN*********", s.id)
}

func (s *Signal) SendNext(v interface{}) {

	for _, obs := range s.Observers {
		obs.SendNext(v)
	}
}

func (s *Signal) SendCompleted() {
	for _, obs := range s.Observers {
		obs.SendCompleted()
	}
	s.Observers = []Observer{}
	s.injector.Close()
}

func (s *Signal) SendFailed(err error) {
	for _, obs := range s.Observers {
		obs.SendFailed(err)
	}
}

func (s *Signal) ListenNext(next NextEvent) {
	// log.Println("add listen next :", next)
	cObs := Observe{next: next}
	s.Observers = append(s.Observers, cObs)
}
func (s *Signal) ListenFailed(failed FailedEvent) {
	cObs := Observe{failed: failed}
	s.Observers = append(s.Observers, cObs)
}
func (s *Signal) ListenCompleted(completed CompletedEvent) {
	cObs := Observe{completed: completed}
	s.Observers = append(s.Observers, cObs)
}

var id = 0

func NewSignal(generator func(obs Injector)) *Signal {
	i := NewChanInjector()
	o := make([]Observer, 0, 0)
	s := &Signal{injector: i, Observers: o}
	s.id = id
	id++
	go s.run()
	go generator(s.injector)
	return s
}

func NewPipe() (*Signal, Injector) {
	i := NewChanInjector()
	o := make([]Observer, 0, 0)
	s := &Signal{injector: i, Observers: o}
	go s.run()
	return s, i
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
