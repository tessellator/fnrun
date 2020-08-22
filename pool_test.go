package fnrun

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewInvokerPool(t *testing.T) {
	t.Run("factory with err returns err", func(t *testing.T) {
		config := InvokerPoolConfig{
			MaxInvokerCount: 5,
			InvokerFactory:  &invokerFactoryThatCannotCreateInvoker{},
			MaxWaitDuration: 5 * time.Millisecond,
			MaxRunnableTime: 0,
		}
		pool, err := NewInvokerPool(config)

		if err != ErrFake {
			t.Errorf("Expected 'fake' err, but got: %+v", err)
		}

		if pool != nil {
			t.Errorf("Expected pool to be nil but got: %+v", pool)
		}
	})

	t.Run("pool contains correct number of invokers", func(t *testing.T) {
		config := InvokerPoolConfig{
			MaxInvokerCount: 5,
			InvokerFactory:  &simpleInvokerFactory{},
			MaxWaitDuration: 5 * time.Millisecond,
			MaxRunnableTime: 0,
		}
		pool, err := NewInvokerPool(config)

		if err != nil {
			t.Fatalf("creating pool got err: %+v", err)
		}

		count := len(pool.invokerChan)
		if count != config.MaxInvokerCount {
			t.Errorf("Expected pool to have %d invokers, but it has %d", config.MaxInvokerCount, count)
		}
	})
}

func TestInvokerPool_Invoke_funcErr(t *testing.T) {
	config := InvokerPoolConfig{
		MaxInvokerCount: 5,
		InvokerFactory:  &errInvokerFactory{},
		MaxWaitDuration: 5 * time.Millisecond,
		MaxRunnableTime: 0,
	}
	pool, err := NewInvokerPool(config)

	if err != nil {
		t.Fatalf("Creating invoker pool returned err: %+v", err)
	}

	input := Input{}

	result, err := pool.Invoke(context.Background(), &input)

	if err != ErrFake {
		t.Errorf("Expected fake error, but got: %+v", err)
	}

	if result != nil {
		t.Errorf("Expected result to be nil, but got: %+v", result)
	}

	length := len(pool.invokerChan)

	if length != config.MaxInvokerCount {
		t.Errorf("Expected available invokers to be %d, but was %d", config.MaxInvokerCount, length)
	}
}

func TestInvokerPool_Invoke_timeout(t *testing.T) {
	config := InvokerPoolConfig{
		MaxInvokerCount: 0,
		InvokerFactory:  &simpleInvokerFactory{},
		MaxWaitDuration: 5 * time.Millisecond,
		MaxRunnableTime: 0,
	}
	pool, err := NewInvokerPool(config)

	if err != nil {
		t.Fatalf("Creating invoker pool returned err: %+v", err)
	}

	input := Input{}

	result, err := pool.Invoke(context.Background(), &input)

	if err != ErrAvailabilityTimeout {
		t.Errorf("Expected availability timeout error, but got: %+v", err)
	}

	if result != nil {
		t.Errorf("Expected result to be nil, but got: %+v", result)
	}
}

func TestInvokerPool_Invoke_success(t *testing.T) {
	config := InvokerPoolConfig{
		MaxInvokerCount: 5,
		InvokerFactory:  &simpleInvokerFactory{},
		MaxWaitDuration: 5 * time.Millisecond,
		MaxRunnableTime: 0,
	}
	pool, err := NewInvokerPool(config)

	if err != nil {
		t.Fatalf("Creating invoker pool returned err: %+v", err)
	}

	input := Input{}

	result, err := pool.Invoke(context.Background(), &input)

	length := len(pool.invokerChan)
	if length != 5 {
		t.Errorf("Expected invokerChan to have 5 elements, but has: %d", length)
	}

	got := string(result.Data)
	want := "some data"

	if got != want {
		t.Errorf("Invoke result data: got %s; want %s", got, want)
	}
}

// -----------------------------------------------------------------------------
// Sample invokers and factories

// ---------------------------------
// Factory that cannot create invoker

var ErrFake = errors.New("fake")

type invokerFactoryThatCannotCreateInvoker struct{}

func (factory *invokerFactoryThatCannotCreateInvoker) NewInvoker() (Invoker, error) {
	return nil, ErrFake
}

// ---------------------------------
// Simple invoker

type simpleInvokerFactory struct{}

func (factory *simpleInvokerFactory) NewInvoker() (Invoker, error) {
	return &simpleInvoker{}, nil
}

type simpleInvoker struct{}

func (sf *simpleInvoker) Invoke(ctx context.Context, input *Input) (*Result, error) {
	return &Result{Data: []byte("some data")}, nil
}

// ---------------------------------
// Invoker that fails on invocation

type errInvokerFactory struct{}

func (factory *errInvokerFactory) NewInvoker() (Invoker, error) {
	return &errInvoker{}, nil
}

type errInvoker struct{}

func (ef *errInvoker) Invoke(context.Context, *Input) (*Result, error) {
	return nil, ErrFake
}
