package fnrun

import (
	"context"
	"errors"
	"io"

	tspb "github.com/golang/protobuf/ptypes"
	"github.com/tessellator/fnrun/fnrun/protobufs"
	"github.com/tessellator/protoio"
)

// ErrMissingTimeout exists to signal that an code is attempting to call an
// invoker without specifying a timeout.
// fnrun assumes that invokers have an associated timeout.
var ErrMissingTimeout = errors.New("Expected a context with a timeout")

// This typing helps avoid key collisions in context.Context; see
// https://blog.golang.org/context#TOC_3.2.

type ctxKey int

const (
	ctxEnvKey ctxKey = iota
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
	Invoke(context.Context, *Input) (*Result, error)
}

// InvokerFactory represents an object that can create instances of Invokers.
type InvokerFactory interface {
	NewInvoker() (Invoker, error)
}

// WithEnv annotates the context with any environment variables the process
// should receive.
func WithEnv(ctx context.Context, env map[string]string) context.Context {
	return context.WithValue(ctx, ctxEnvKey, env)
}

// Env retrieves the environment variables placed on the Invoker's context.
// The second argument is false if there are no environment variables
// associated with the context.
func Env(ctx context.Context) (map[string]string, bool) {
	env, hasEnv := ctx.Value(ctxEnvKey).(map[string]string)
	return env, hasEnv
}

// WriteTo writes the ExecutionContext to the specified writer.
func WriteTo(ctx context.Context, w io.Writer) (int64, error) {
	envVars := []*protobufs.EnvironmentVariable{}
	env, hasEnv := Env(ctx)
	if hasEnv {
		for k, v := range env {
			envVars = append(envVars, &protobufs.EnvironmentVariable{Name: k, Value: v})
		}
	}

	stopTime, hasTimeout := ctx.Deadline()
	if !hasTimeout {
		return 0, ErrMissingTimeout
	}
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
