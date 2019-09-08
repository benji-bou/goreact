package goreact

import (
	uuid "github.com/satori/go.uuid"
)

type NextEvent func(v interface{}) Disposable
type FailedEvent func(err error) Disposable
type CompletedEvent func(completed bool) Disposable
type TerminatedEvent func() Disposable

type NextEventWithMeta func(v interface{}, metas ...interface{}) Disposable
type FailedEventWithMeta func(err error, metas ...interface{}) Disposable
type CompletedEventWithMeta func(completed bool, metas ...interface{}) Disposable

type Observer interface {
	Injector
	GetId() uuid.UUID
	CleanUp()
	SendTerminated()
}

type Observe struct {
	id         uuid.UUID
	next       NextEvent
	failed     FailedEvent
	completed  CompletedEvent
	terminated TerminatedEvent
	timeline   BagDisposer
}

func (o Observe) GetId() uuid.UUID {
	return o.id
}

func (o Observe) CleanUp() {
	o.timeline.Dispose()
}

func (o Observe) SendFailed(err error) {
	if o.failed != nil {
		disp := o.failed(err)
		if disp != nil {
			o.timeline.Add(disp)
		}
	}
}

func (o Observe) SendNext(value interface{}) {
	if o.next != nil {
		disp := o.next(value)
		if disp != nil {
			o.timeline.Add(disp)
		}
	}

}

func (o Observe) SendCompleted() {
	if o.completed != nil {
		disp := o.completed(true)
		if disp != nil {
			o.timeline.Add(disp)
		}
	}
}
func (o Observe) SendTerminated() {
	if o.terminated != nil {
		disp := o.terminated()
		if disp != nil {
			o.timeline.Add(disp)
		}
	}
	// o.CleanUp()
}

type MetaObserve struct {
	id         uuid.UUID
	next       NextEventWithMeta
	failed     FailedEventWithMeta
	completed  CompletedEventWithMeta
	terminated TerminatedEvent
	timeline   BagDisposer
	metas      []interface{}
}

func (o MetaObserve) GetId() uuid.UUID {
	return o.id
}

func (o MetaObserve) CleanUp() {
	o.timeline.Dispose()
}

func (o MetaObserve) SendFailed(err error) {
	if o.failed != nil {
		disp := o.failed(err, o.metas...)
		if disp != nil {
			o.timeline.Add(disp)
		}
	}
}

func (o MetaObserve) SendNext(value interface{}) {
	if o.next != nil {
		disp := o.next(value, o.metas...)
		if disp != nil {
			o.timeline.Add(disp)
		}
	}
}

func (o MetaObserve) SendCompleted() {
	if o.completed != nil {
		disp := o.completed(true, o.metas...)
		if disp != nil {
			o.timeline.Add(disp)
		}
	}
}

func (o MetaObserve) SendTerminated() {
	if o.terminated != nil {
		disp := o.terminated()
		if disp != nil {
			o.timeline.Add(disp)
		}
	}
	// o.CleanUp()
}
