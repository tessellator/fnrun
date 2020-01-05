package fnrun

import (
	"errors"
	"time"
)

// InvokerPool represents a pool of Invoker workers that can be used to handle
// invocation requests.
//
// The pool will attempt to satisfy the requirements specified by the config
// object, including maintaining a number of invoker instances available to
// handle invocations; if an Invoker fails, it is discarded and replaced by a
// new Invoker instance.
type InvokerPool struct {
	config      InvokerPoolConfig
	invokerChan chan Invoker
}

// InvokerPoolConfig contains the configuration data for an InvokerPool.
type InvokerPoolConfig struct {
	MaxInvokerCount int
	InvokerFactory  InvokerFactory
	MaxWaitDuration time.Duration
}

// NewInvokerPool creats a new InvokerPool with the provided configuration.
func NewInvokerPool(config InvokerPoolConfig) (*InvokerPool, error) {
	invokerChan := make(chan Invoker, config.MaxInvokerCount)
	for i := 0; i < config.MaxInvokerCount; i++ {
		invoker, err := config.InvokerFactory.NewInvoker()
		if err != nil {
			return nil, err
		}
		invokerChan <- invoker
	}

	pool := &InvokerPool{
		config:      config,
		invokerChan: invokerChan,
	}

	return pool, nil
}

// Invoke attempts to use an Invoker in the pool to satisfy the invocation
// request.
//
// If a worker Invoker is not available within the MaxWaitDuration of the pool
// configuration, an ErrAvailabilityTimeout error is returned from this
// function.
func (pool *InvokerPool) Invoke(input *Input, ctx *ExecutionContext) (*Result, error) {
	// TODO Keep track of how many invoker instances we have (that are in use or
	// available; that haven't failed and been unreplaced). Once that number hits
	// zero, we should return an appropriate error
	select {
	case invoker := <-pool.invokerChan:
		result, err := invoker.Invoke(input, ctx)
		if err != nil {
			newInvoker, factoryErr := pool.config.InvokerFactory.NewInvoker()
			if factoryErr != nil {
				return nil, errors.New("could not create invoker")
			}
			pool.invokerChan <- newInvoker
			return nil, err
		}
		pool.invokerChan <- invoker
		return result, err
	case <-time.After(pool.config.MaxWaitDuration):
		return nil, ErrAvailabilityTimeout
	}
}

// ErrAvailabilityTimeout is an error that indicates that an invoker did not
// become available within the allow time period.
var ErrAvailabilityTimeout = errors.New("could not get access to invoker before timeout")
