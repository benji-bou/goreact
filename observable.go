package goreact

import (
	"context"

	"github.com/benji-bou/chantools"
)

type Mappable[I any, O any] interface {
	Map(mapper func (elem I) O) Observable[O]
}

type Filterable[T any] interface {

	Filter(filter func (elem T) bool)
	
}

type Mergeable[T any] interface {
	Merge(observer ...Observable[T])
}


type Observable[T any] interface {
	Subscribe(ctx context.Context, next func(T), err func(err error)) context.CancelFunc 
}

type ObserveStream[T any] struct {
	input <-chan T
	err <-chan error
} 

func (obs ObserveStream[T])Subscribe(ctx context.Context, next func(T), err func(err error)) context.CancelFunc {
	ctxSubscription, cancelFunc := context.WithCancel(ctx)
	go func(ctxSubscription context.Context, next func(T), err func(err error)) {
		for {
			select {
			case i, isOk := <- obs.input:
				if !isOk  {
					return
				}
				next(i)
			case e, isOk := <- obs.err:
				if !isOk  {
					return
				}
				err(e)
			case <- ctxSubscription.Done():
				return
			}
		}
	
	}(ctxSubscription, next, err)
	return cancelFunc
}

func (obs ObserveStream[T]) ToMap() Mappable {

}

func Observe[T any](signal func(channel chan<- T, err chan<- error)) Observable[T]  {
	i, e := chantools.ChanErrGenerator(signal)
	return ObserveStream[T]{input:i, err: e}
}