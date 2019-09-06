// Package timeout is primarily intended as a test utility. It provides the
// function timeout.After(ms, wait) which will wait ms number of milliseconds
// for the wait object.
//
// The wait value can be a function. It will be called with no arguments. If it
// does not return before the timeout duration a TimeoutError is returned. If
// the last return value of the function is of type error and the function does
// not timeout but does return an error, that error will be returned.
//
// The wait value can be a channel. If it is a send only channel, the zero value
// for the channel will be sent. If it blocks longer than the timeout duration,
// a TimeoutError is returned. If the channel can receive, it will try for the
// timeout duration. If it does not receive within the duration, a TimeoutError
// is returned. If the channel does receive but the value that comes through the
// channel is an interface that fulfills error and is not nil, that error value
// is returned.
//
// The wait value can be a *sync.WaitGroup. It must be a pointer to a WaitGroup,
// passing in a WaitGroup by value causes it's Wait() method to not behave
// correctly.
//
// If the wait value is not a valid type an InvalidWait error is returned.
package timeout

import (
	"reflect"
	"sync"
	"time"
)

var (
	errType = reflect.TypeOf((*error)(nil)).Elem()
	wgType  = reflect.TypeOf((*sync.WaitGroup)(nil))
)

const (
	// ErrorMsg returned by timeout.Error.Error()
	ErrorMsg = "Timeout"
	// InvalidWaitMsg returned by timeout.InvalidWaitMsg.Error()
	InvalidWaitMsg = "Argument 'wait' is not a valid type. Must be a function, channel or *sync.WaitGroup"
)

// Error indicates that a timeout occurred.
type Error struct{}

// Error fulfills error.
func (Error) Error() string { return ErrorMsg }

// InvalidWait indicates that the wait argument in the After function was not
// a valid type.
type InvalidWait struct{}

func (InvalidWait) Error() string { return InvalidWaitMsg }

// After returns Timeout when a specified number of milliseconds (ms) have
// passed if wait has not completed. If wait is not a valid type InvalidWait is
// returned.
func After(ms int, wait interface{}) error {
	d := time.Millisecond * time.Duration(ms)
	v := reflect.ValueOf(wait)
	switch v.Kind() {
	case reflect.Chan:
		if v.Type().ChanDir() == reflect.SendDir {
			return chSend(d, v)
		}
		return chRecv(d, v)
	case reflect.Func:
		return fn(d, v)
	}
	if v.Type() == wgType {
		return wg(d, wait.(*sync.WaitGroup))
	}
	return InvalidWait{}
}

func chSend(d time.Duration, v reflect.Value) error {
	i, _, _ := reflect.Select([]reflect.SelectCase{
		{
			Dir:  reflect.SelectSend,
			Chan: v,
			Send: reflect.Zero(v.Type().Elem()),
		},
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(time.After(d)),
		},
	})
	if i == 1 {
		return Error{}
	}
	return nil
}

func chRecv(d time.Duration, v reflect.Value) error {
	i, r, _ := reflect.Select([]reflect.SelectCase{
		{
			Dir:  reflect.SelectRecv,
			Chan: v,
		},
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(time.After(d)),
		},
	})
	if i == 1 {
		return Error{}
	}
	if r.Kind() == reflect.Interface && r.Type().Implements(errType) && !r.IsNil() {
		return r.Interface().(error)
	}
	return nil
}

func fn(d time.Duration, v reflect.Value) (err error) {
	ch := make(chan []reflect.Value)
	go func() {
		ch <- v.Call(nil)
	}()
	select {
	case <-time.After(d):
		err = Error{}
	case out := <-ch:
		if ln := len(out); ln > 0 {
			if v := out[ln-1]; v.Kind() == reflect.Interface && v.Type().Implements(errType) && !v.IsNil() {
				return v.Interface().(error)
			}
		}
	}
	return
}

func wg(d time.Duration, wg *sync.WaitGroup) (err error) {
	ch := make(chan struct{})
	go func() {
		wg.Wait()
		ch <- struct{}{}
	}()
	select {
	case <-time.After(d):
		err = Error{}
	case <-ch:
	}
	return
}
