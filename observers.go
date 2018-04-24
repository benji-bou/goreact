package goreact

import (
	uuid "github.com/satori/go.uuid"
)

type Observe struct {
	id        uuid.UUID
	next      NextEvent
	failed    FailedEvent
	completed CompletedEvent
}

func (o Observe) GetId() uuid.UUID {
	return o.id
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

type MetaObserve struct {
	id        uuid.UUID
	next      NextEventWithMeta
	failed    FailedEventWithMeta
	completed CompletedEventWithMeta

	metas []interface{}
}

func (o MetaObserve) GetId() uuid.UUID {
	return o.id
}

func (o MetaObserve) SendFailed(err error) {
	if o.failed != nil {
		o.failed(err, o.metas...)
	}
}

func (o MetaObserve) SendNext(value interface{}) {
	if o.next != nil {
		o.next(value, o.metas...)
	}
}

func (o MetaObserve) SendCompleted() {
	if o.completed != nil {
		o.completed(true, o.metas...)
	}
}
