package goreact

import (
	uuid "github.com/satori/go.uuid"
)

type NextEvent func(v interface{}) Disposable
type FailedEvent func(err error) Disposable
type CompletedEvent func(completed bool) Disposable

type NextEventWithMeta func(v interface{}, metas ...interface{}) Disposable
type FailedEventWithMeta func(err error, metas ...interface{}) Disposable
type CompletedEventWithMeta func(completed bool, metas ...interface{}) Disposable

type Observer interface {
	Injector
	GetId() uuid.UUID
}

type Observe struct {
	id        uuid.UUID
	next      NextEvent
	failed    FailedEvent
	completed CompletedEvent
	timeline  BagDisposer
}

func (o Observe) GetId() uuid.UUID {
	return o.id
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

type MetaObserve struct {
	id        uuid.UUID
	next      NextEventWithMeta
	failed    FailedEventWithMeta
	completed CompletedEventWithMeta
	timeline  BagDisposer
	metas     []interface{}
}

func (o MetaObserve) GetId() uuid.UUID {
	return o.id
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
