package fnrun

import (
	"context"
	"io"
	"os/exec"
	"time"

	"github.com/tessellator/executil"
)

type cmdInvoker struct {
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	stdout          io.ReadCloser
	maxRunnableTime time.Duration
}

// NewCmdInvoker creates an object that can invoke the provided exec.Cmd.
//
// This function assumes control of the cmd, and it is the responsibility of the
// caller to ensure that the cmd is not used after being provided to this
// function.
//
// This object kills the OS process managed by the provided cmd when an
// invocation fails, so the object returned from this function should not be
// reused if a call to Invoke returns an error.
func NewCmdInvoker(cmd *exec.Cmd) (Invoker, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	p := &cmdInvoker{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
	}

	return p, nil
}

func (cf *cmdInvoker) Invoke(ctx context.Context, input *Input) (*Result, error) {
	if _, hasTimeout := ctx.Deadline(); !hasTimeout {
		return nil, ErrMissingTimeout
	}

	err := cf.cmd.Start()
	if err != nil {
		return nil, err
	}

	_, err = input.WriteTo(cf.stdin)
	if err != nil {
		cf.cmd.Process.Kill()
		return nil, err
	}

	_, err = WriteTo(ctx, cf.stdin)
	if err != nil {
		cf.cmd.Process.Kill()
		return nil, err
	}

	resultChan := make(chan *Result, 1)
	errChan := make(chan error, 1)

	go func() {
		result := &Result{}
		err := ReadFrom(cf.stdout, result)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	select {
	case response := <-resultChan:
		return response, nil
	case <-ctx.Done():
		cf.cmd.Process.Kill()
		return nil, ctx.Err()
	case err = <-errChan:
		cf.cmd.Process.Kill()
		return nil, err
	}
}

type cmdInvokerFactory struct {
	cmd *exec.Cmd
}

// NewCmdInvokerFactory creates a factory that can create new instances of
// CmdInvoker intances.
//
// The cmd will be cloned for each new instances, which means that multiple
// calls to the factory can create multiple copies of OS processes.
func NewCmdInvokerFactory(cmd *exec.Cmd) InvokerFactory {
	return &cmdInvokerFactory{cmd: cmd}
}

func (factory *cmdInvokerFactory) NewInvoker() (Invoker, error) {
	newCmd := executil.CloneCmd(factory.cmd)
	return NewCmdInvoker(newCmd)
}
