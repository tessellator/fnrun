package fnrun

import (
	"bufio"
	"bytes"
	"testing"
	"time"

	"github.com/tessellator/fnrun/fnrun/protobufs"
	"github.com/tessellator/protoio"
)

func TestExecutionContext_WriteTo(t *testing.T) {
	vars := make(map[string]string)
	vars["hello"] = "world"

	ctx := ExecutionContext{
		MaxRunnableTime: 30 * time.Second,
		Env:             vars,
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	_, err := ctx.WriteTo(w)
	if err != nil {
		t.Fatalf("WriteTo() got err: %+v", err)
	}

	w.Flush()

	r := bufio.NewReader(&buf)

	pctx := protobufs.ExecutionContext{}
	err = protoio.Read(r, &pctx)

	if err != nil {
		t.Errorf("Read() returned err: %+v", err)
	}

	v := pctx.GetEnvVars()[0]
	name := v.GetName()
	val := v.GetValue()

	if name != "hello" {
		t.Errorf("Expected var name 'hello', but got %s", name)
	}

	if val != "world" {
		t.Errorf("Expected var value 'world', but got %s", val)
	}
}
