package goreact

import (
	"log"

	uuid "github.com/satori/go.uuid"
)

// "log"

type Injector interface {
	SendFailed(err error)
	SendNext(value interface{})
	SendCompleted()
}

type Disposable interface {
	Dispose()
}

type DisposableFunc func()

func (df DisposableFunc) Dispose() {
	df()
}

type ReusableBagDisposer struct {
	disposers DisposerList
	add       chan Disposable
	dispose   chan struct{}
}

type DisposerWithAction struct {
	Disposable
	action func()
}

func MakeDisposerWithAction(s *Signal, action func()) DisposerWithAction {
	disp, _ := makeDisposer(s)
	return DisposerWithAction{Disposable: disp, action: action}
}

func (dwa DisposerWithAction) Dispose() {
	dwa.action()
	dwa.Disposable.Dispose()
}

func MakeReusableBagDisposer() ReusableBagDisposer {
	bag := ReusableBagDisposer{disposers: DisposerList{}, add: make(chan Disposable), dispose: make(chan struct{})}
	go func(bag ReusableBagDisposer) {
		for {
			select {
			case disp := <-bag.add:
				bag.disposers = append(bag.disposers, disp)
			case <-bag.dispose:
				bag.disposers.Dispose()
				bag.disposers = DisposerList{}
			}
		}
	}(bag)
	return bag
}

func (bag ReusableBagDisposer) Dispose() {
	bag.dispose <- struct{}{}
}

func (bag ReusableBagDisposer) Add(disps ...Disposable) {
	for _, disp := range disps {
		if disp != nil {
			bag.add <- disp
		}
	}
}

type BagDisposer struct {
	ReusableBagDisposer
	hasBeenDisposed bool
}

func MakeBagDisposer() BagDisposer {
	singleBag := BagDisposer{ReusableBagDisposer: ReusableBagDisposer{disposers: DisposerList{}, add: make(chan Disposable), dispose: make(chan struct{})}, hasBeenDisposed: false}
	go func(sbag BagDisposer) {
		for {
			select {
			case disp := <-sbag.add:
				if sbag.hasBeenDisposed == false {
					sbag.disposers = append(sbag.disposers, disp)
				}
			case <-sbag.dispose:
				sbag.disposers.Dispose()
				sbag.disposers = DisposerList{}
				sbag.hasBeenDisposed = true
			}
		}
	}(singleBag)
	return singleBag
}

type DisposerList []Disposable

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

func makeDisposer(sig *Signal) (BagDisposer, uuid.UUID) {
	id, _ := uuid.NewV1()
	disp := Disposer{id: id, unsubscribe: sig.unsubscribe}
	bag := MakeBagDisposer()
	bag.Add(disp)
	return bag, id
}

type Signal struct {
	// isCompleted bool
	id          uint64
	injector    *ChanInjector
	observers   map[uuid.UUID]Observer
	subscribe   chan Observer
	unsubscribe chan uuid.UUID
	done        chan struct{}
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
			break L
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
	s.sendTerminated()
	s.observers = map[uuid.UUID]Observer{}
	log.Println("*******END OF RUN*********", s.id)
}

func (s *Signal) stopSignal() {
	s.injector.close()
}

func (s *Signal) sendTerminated() {
	for _, obs := range s.observers {
		obs.SendTerminated()
	}
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
	s.observers = map[uuid.UUID]Observer{}
}

func (s *Signal) ListenNext(next NextEvent) Disposable {
	// log.Println("add listen next :", next)
	disp, id := makeDisposer(s)
	cObs := Observe{next: next, id: id, timeline: disp}
	s.subscribe <- cObs
	return disp
}
func (s *Signal) ListenFailed(failed FailedEvent) Disposable {
	disp, id := makeDisposer(s)
	cObs := Observe{failed: failed, id: id, timeline: disp}
	s.subscribe <- cObs
	return disp

}
func (s *Signal) ListenCompleted(completed CompletedEvent) Disposable {
	disp, id := makeDisposer(s)
	cObs := Observe{completed: completed, id: id, timeline: disp}
	s.subscribe <- cObs
	return disp
}

func (s *Signal) ListenTerminated(terminated TerminatedEvent) Disposable {
	disp, id := makeDisposer(s)
	cObs := Observe{terminated: terminated, id: id, timeline: disp}
	s.subscribe <- cObs
	return disp
}

func (s *Signal) Listen(next NextEvent, failed FailedEvent, completed CompletedEvent) Disposable {
	disp, id := makeDisposer(s)
	cObs := Observe{next: next, failed: failed, completed: completed, id: id, timeline: disp}
	s.subscribe <- cObs
	return disp
}

func (s *Signal) ListenNextWithMeta(next NextEventWithMeta, metas ...interface{}) Disposable {
	// log.Println("add listen next :", next)
	disp, id := makeDisposer(s)
	cObs := MetaObserve{next: next, id: id, metas: metas, timeline: disp}
	s.subscribe <- cObs
	return disp
}
func (s *Signal) ListenFailedWithMeta(failed FailedEventWithMeta, metas ...interface{}) Disposable {
	disp, id := makeDisposer(s)
	cObs := MetaObserve{failed: failed, id: id, metas: metas, timeline: disp}

	s.subscribe <- cObs
	return disp

}
func (s *Signal) ListenCompletedWithMeta(completed CompletedEventWithMeta, metas ...interface{}) Disposable {
	disp, id := makeDisposer(s)
	cObs := MetaObserve{completed: completed, id: id, metas: metas, timeline: disp}
	s.subscribe <- cObs
	return disp

}

func (s *Signal) ListenWithMeta(next NextEventWithMeta, failed FailedEventWithMeta, completed CompletedEventWithMeta, metas ...interface{}) Disposable {
	disp, id := makeDisposer(s)
	cObs := MetaObserve{next: next, failed: failed, completed: completed, id: id, metas: metas, timeline: disp}
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

func NewSignalWithMeta(generator func(obs Injector, meta ...interface{}), meta ...interface{}) *Signal {
	s := NewEmptySignal()
	go func(generator func(obs Injector, meta ...interface{}), s *Signal, meta ...interface{}) {
		generator(s.injector, meta...)
		s.stopSignal()
	}(generator, s, meta)
	return s
}

func NewSignal(generator func(obs Injector)) *Signal {
	s := NewEmptySignal()
	go func(generator func(obs Injector), s *Signal) {
		generator(s.injector)
		s.stopSignal()
	}(generator, s)
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

func (i *ChanInjector) close() {
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
		s.ListenNext(func(nextValue interface{}) Disposable {
			return next(nextValue)
		})
	}
	if failed != nil {
		s.ListenFailed(func(err error) Disposable {
			return failed(err)
		})
	}
	if completed != nil {
		s.ListenCompleted(func(completedValue bool) Disposable {
			return completed(completedValue)
		})
	}
	return s
}

func (s *Signal) OnNext(side NextEvent) *Signal {
	return s.On(side, nil, nil)
}

func (s *Signal) Map(transformer func(value interface{}) (interface{}, error)) *Signal {
	pipe := NewPipe()
	s.ListenNext(func(next interface{}) Disposable {
		newValue, err := transformer(next)
		if err != nil {
			pipe.Injector.SendFailed(err)
			return nil
		}
		pipe.Injector.SendNext(newValue)
		return nil
	})
	s.ListenFailed(func(err error) Disposable {
		pipe.Injector.SendFailed(err)
		return nil
	})
	s.ListenCompleted(func(completed bool) Disposable {
		pipe.Injector.SendCompleted()
		return nil
	})
	return pipe.Signal
}

func Merge(inners ...*Signal) *Signal {
	pipe := NewPipe()
	count := len(inners)
	var bag BagDisposer = MakeBagDisposer()
	for _, inner := range inners {
		sig := inner.Listen(func(event interface{}) Disposable {
			pipe.Injector.SendNext(event)
			return nil
		}, func(err error) Disposable {
			pipe.Injector.SendFailed(err)
			return nil
		}, func(completed bool) Disposable {
			count--
			if count == 0 {
				pipe.Injector.SendCompleted()
			}
			return nil
		})
		bag.Add(sig)
	}
	return pipe.Signal
}

func (s *Signal) Merge(inners ...*Signal) *Signal {
	all := append(inners, s)
	return Merge(all...)
}

func SafelyListeningSignals(s *Signal, action func(bag BagDisposer)) {
	done := make(chan struct{})
	bag := MakeBagDisposer()
	dispTerminated := s.ListenTerminated(func() Disposable {
		done <- struct{}{}
		return nil
	})
	bag.Add(dispTerminated)
	action(bag)
	<-done
	bag.Dispose()

}

func (s *Signal) Filter(filter func(value interface{}) (include bool)) *Signal {
	return NewSignal(func(obs Injector) {
		SafelyListeningSignals(s, func(bag BagDisposer) {
			disp := s.Listen(func(next interface{}) Disposable {
				if filter(next) == true {
					obs.SendNext(next)
				}
				return nil
			}, func(err error) Disposable {
				obs.SendFailed(err)
				return nil
			}, func(completed bool) Disposable {
				obs.SendCompleted()
				return nil
			})
			bag.Add(disp)
		})
	})

}

func (s *Signal) FlatMap(mapping func(value interface{}) *Signal) *Signal {

	return NewSignal(func(obs Injector) {
		done := make(chan int)
		count := 0
		dispMain := s.ListenNext(func(next interface{}) Disposable {
			mapped := mapping(next)
			count++

			disp := mapped.Listen(func(v interface{}) Disposable {
				obs.SendNext(v)
				return nil
			}, func(err error) Disposable {
				obs.SendFailed(err)
				done <- 0
				return nil
			}, func(completed bool) Disposable {
				count--
				done <- count
				return nil
			})
			return disp
		})
		dispMainFailed := s.ListenFailed(func(err error) Disposable {
			obs.SendFailed(err)
			done <- 0
			return nil
		})
		for nbSignalRemain := range done {
			if nbSignalRemain == 0 {
				dispMain.Dispose()
				dispMainFailed.Dispose()

			}
		}
	})

}
