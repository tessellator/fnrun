package fnrun

import (
	"bufio"
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/tessellator/fnrun/fnrun/protobufs"
	"github.com/tessellator/protoio"
)

func TestExecutionContext_WriteTo(t *testing.T) {
	vars := make(map[string]string)
	vars["hello"] = "world"

	ctx := WithEnv(context.Background(), vars)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	_, err := WriteTo(ctx, w)
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
