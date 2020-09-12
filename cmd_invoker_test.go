package fnrun

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/tessellator/fnrun/fnrun/protobufs"
	protoio "github.com/tessellator/protoio"
)

func TestNewCmdInvoker(t *testing.T) {
	t.Run("with an executable that does not exist", func(t *testing.T) {
		cmd := exec.Command("does_not_exist")

		_, err := NewCmdInvoker(cmd)

		if err == nil {
			t.Errorf("NewCmdInvoker() did not return error")
		}
	})
}

func TestCmdInvoker_Invoke_crash(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=Test_SleepySubprocess")
	cmd.Env = append(os.Environ(), "GO_RUNNING_SUBPROCESS=1")
	invoker, err := NewCmdInvoker(cmd)

	if err != nil {
		t.Fatalf("NewCmdInvoker() returned error: %+v", err)
	}

	input := Input{}
	result, err := invoker.Invoke(context.Background(), &input)

	if err == nil {
		t.Error("Invoke(): did not receive error but expected to")
	}

	if result != nil {
		t.Errorf("Invoke(): expected result to be nil but got value: %+v", result)
	}
}

func TestCmdInvoker_Invoke_runTooLong(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=Test_SleepySubprocess")
	cmd.Env = append(os.Environ(), "GO_RUNNING_SUBPROCESS=1")
	invoker, err := NewCmdInvoker(cmd)

	if err != nil {
		t.Fatalf("NewCmdInvoker() returned error: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	input := Input{Data: []byte("some data")}

	_, err = invoker.Invoke(ctx, &input)

	if err != context.DeadlineExceeded {
		t.Errorf("Expected deadline exceeded error but got: %+v", err)
	}
}

func TestCmdInvoker_Invoke_invalidReturn(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=Test_WriteEventSubprocess")
	cmd.Env = append(os.Environ(), "GO_RUNNING_SUBPROCESS=1")
	invoker, err := NewCmdInvoker(cmd)
	if err != nil {
		t.Fatalf("NewCmdInvoker() unexpectedly returned error: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	input := Input{Data: []byte("some data")}

	result, err := invoker.Invoke(ctx, &input)

	if err == nil {
		t.Errorf("Expected Invoke() to return an error, but it did not")
	}

	if result != nil {
		t.Errorf("Invoke(): did not expect result, but got: %+v", result)
	}
}

func TestCmdInvoker_Invoke_closeNoReturn(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=Test_WriteNoOutput")
	cmd.Env = append(os.Environ(), "GO_RUNNING_SUBPROCESS=1")
	invoker, err := NewCmdInvoker(cmd)
	if err != nil {
		t.Fatalf("NewCmdInvoker() unexpectedly returned error: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	input := Input{Data: []byte("some data")}

	result, err := invoker.Invoke(ctx, &input)

	if err == nil {
		t.Errorf("Expected Invoke() to return an error, but it did not")
	}

	if result != nil {
		t.Errorf("Invoke(): did not expect result, but got: %+v", result)
	}
}

func TestCmdInvoker_Invoke_validReturn(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=Test_GreetingSubprocess")
	cmd.Env = append(os.Environ(), "GO_RUNNING_SUBPROCESS=1")

	invoker, err := NewCmdInvoker(cmd)

	if err != nil {
		t.Fatalf("NewCmdInvoker() returned error: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Hour)
	defer cancel()

	input := Input{Data: []byte("world")}
	result, err := invoker.Invoke(ctx, &input)

	if err != nil {
		t.Errorf("Invoke() returned err: %+v", err)
	}

	want := "Hello, world!"
	got := string(result.Data)

	if want != got {
		t.Errorf("Did not read expected result: got %s; want %s", got, want)
	}
}

// -----------------------------------------------------------------------------
// Following are various subprocesses used for testing. Each is named according
// to its behavior.

func Test_CrashingSubprocess(t *testing.T) {
	if os.Getenv("GO_RUNNING_SUBPROCESS") != "1" {
		return
	}

	os.Exit(1)
}

func Test_SleepySubprocess(t *testing.T) {
	if os.Getenv("GO_RUNNING_SUBPROCESS") != "1" {
		return
	}

	<-time.After(200 * time.Millisecond)

	result := protobufs.Result{}
	protoio.Write(os.Stdout, &result)
}

func Test_WriteEventSubprocess(t *testing.T) {
	if os.Getenv("GO_RUNNING_SUBPROCESS") != "1" {
		return
	}

	event := protobufs.Event{Data: []byte("this is an event")}
	protoio.Write(os.Stdout, &event)
}

func Test_WriteNoOutput(t *testing.T) {
	if os.Getenv("GO_RUNNING_SUBPROCESS") != "1" {
		return
	}

	// do nothing!
}

func Test_GreetingSubprocess(t *testing.T) {
	if os.Getenv("GO_RUNNING_SUBPROCESS") != "1" {
		return
	}

	ctx := protobufs.ExecutionContext{}
	event := protobufs.Event{}

	protoio.Read(os.Stdin, &event)
	protoio.Read(os.Stdin, &ctx)

	response := "Hello, " + string(event.GetData()) + "!"
	result := protobufs.Result{Data: []byte(response)}
	protoio.Write(os.Stdout, &result)
}
