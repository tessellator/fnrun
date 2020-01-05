package fnrun

import (
	"errors"
	"io"
	"time"

	tspb "github.com/golang/protobuf/ptypes"
	"github.com/tessellator/fnrun/fnrun/protobufs"
	"github.com/tessellator/protoio"
)

// Invoker represents something that can be called with an input and context
// and return a result.
//
// It is similar to a function but the underlying implementation may be
// different.
type Invoker interface {
	// Invoke triggers a call to the underlying implementation with the specified
	// input and context.
	//
	// The call may time out out subject to the maximum runnable time defined in
	// ctx.
	//
	// If an error is returned, the instance cannot be guaranteed to be invokable
	// again, subject to implementation.
	Invoke(*Input, *ExecutionContext) (*Result, error)
}

// InvokerFactory represents an object that can create instances of Invokers.
type InvokerFactory interface {
	NewInvoker() (Invoker, error)
}

// ExecutionContext contains contextual information for an invocation.
type ExecutionContext struct {
	MaxRunnableTime time.Duration
	Env             map[string]string
}

// WriteTo writes the ExecutionContext to the specified writer.
func (ctx *ExecutionContext) WriteTo(w io.Writer) (int64, error) {
	envVars := []*protobufs.EnvironmentVariable{}
	for k, v := range ctx.Env {
		envVars = append(envVars, &protobufs.EnvironmentVariable{Name: k, Value: v})
	}

	stopTime := time.Now().Add(ctx.MaxRunnableTime)
	stopTimeProto, err := tspb.TimestampProto(stopTime)
	if err != nil {
		return 0, err
	}

	protoCtx := protobufs.ExecutionContext{
		EnvVars:  envVars,
		StopTime: stopTimeProto,
	}

	return protoio.Write(w, &protoCtx)
}

// Input contains the data that is passed to an invocation.
//
// The data is represented as a byte array to be as generic as possible across
// many implementations.
type Input struct {
	Data []byte
}

// WriteTo writes the Input to the specified writer.
func (input *Input) WriteTo(w io.Writer) (int64, error) {
	pInput := protobufs.Event{
		Data: input.Data,
	}

	return protoio.Write(w, &pInput)
}

// Result represents the result of an invocation.
//
// The result includes a status, data, and additional env information.
//
// The status is an int and can represent something meaningful to the code
// calling Invoke, such as an HTTP status code or an OS process return code.
//
// The data returns is a byte array so as to be as agnostic to the underlying
// implementation details as possible.
//
// The env should be any environmental data that should be considered important
// to the return of the code. For example, this could contain HTTP header
// name/value pairs.
type Result struct {
	Status int
	Data   []byte
	Env    map[string]string
}

// ReadFrom reads a Result from the specified reader and populates the specified
// result.
func ReadFrom(r io.Reader, result *Result) error {
	pResult := protobufs.Result{}
	err := protoio.Read(r, &pResult)
	if err != nil {
		return err
	}

	env := make(map[string]string)
	for _, envVar := range pResult.GetEnvVars() {
		env[envVar.GetName()] = envVar.GetValue()
	}

	result.Status = int(pResult.GetStatus())
	result.Data = pResult.GetData()
	result.Env = env

	return nil
}

// ErrExecutionTimeout is an error that indicates the allowable execution time
// was exceeded.
var ErrExecutionTimeout = errors.New("execution time exceeded")
