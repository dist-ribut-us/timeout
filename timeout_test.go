package timeout_test

import (
	"errors"
	"github.com/dist-ribut-us/timeout"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestChan(t *testing.T) {
	intCh := make(chan int)
	go func() {
		intCh <- 10
	}()

	err := timeout.After(2, intCh)
	assert.NoError(t, err)

	err = timeout.After(2, intCh)
	assert.Equal(t, timeout.Error{}, err)

	err = timeout.After(2, chan<- int(intCh))
	assert.Equal(t, timeout.Error{}, err)

	go func() {
		assert.Equal(t, 0, <-intCh)
	}()
	err = timeout.After(2, chan<- int(intCh))
	assert.NoError(t, err)

	errCh := make(chan error)
	go func() {
		errCh <- nil
	}()

	err = timeout.After(2, errCh)
	assert.NoError(t, err)

	err = timeout.After(2, errCh)
	assert.Equal(t, timeout.Error{}, err)

	go func() {
		errCh <- errors.New("testing")
	}()
	err = timeout.After(2, errCh)
	assert.Equal(t, "testing", err.Error())
}

func TestFunc(t *testing.T) {
	err := timeout.After(2, func() {})
	assert.NoError(t, err)

	err = timeout.After(2, func() {
		time.Sleep(time.Millisecond * 5)
	})
	assert.Equal(t, timeout.Error{}, err)

	err = timeout.After(2, func() error {
		return errors.New("testing")
	})
	assert.Equal(t, "testing", err.Error())
}

func TestWaitGroup(t *testing.T) {
	wg := &sync.WaitGroup{}

	err := timeout.After(2, wg)
	assert.NoError(t, err)

	wg.Add(1)
	go func() {
		time.Sleep(time.Millisecond)
		wg.Done()
	}()
	err = timeout.After(4, wg)
	assert.NoError(t, err)

	wg.Add(1)
	err = timeout.After(4, wg)
	assert.Equal(t, timeout.Error{}, err)
}

func TestErrors(t *testing.T) {
	err := timeout.After(10, 3.1415)
	assert.Equal(t, timeout.InvalidWait{}, err)

	assert.Equal(t, timeout.ErrorMsg, timeout.Error{}.Error())
	assert.Equal(t, timeout.InvalidWaitMsg, timeout.InvalidWait{}.Error())
}
