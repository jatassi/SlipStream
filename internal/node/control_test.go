package node

import (
	"errors"
	"testing"

	"github.com/jatassi/slipstream/internal/node/op"
)

func TestStopNodeReturnsOnceTheDataDirIsFree(t *testing.T) {
	env, _ := testEnv(t)
	n := startTest(t, env)
	if err := StopNode(t.Context(), env.DataDir); err != nil {
		t.Fatalf("stop: %v", err)
	}
	startTest(t, env)
	<-n.Done()
	if n.Err() != nil {
		t.Fatalf("node stopped with %v", n.Err())
	}
}

func TestUnknownOpNamesTheNodeVersion(t *testing.T) {
	env, _ := testEnv(t)
	startTest(t, env)
	rename := op.Op[struct{}, struct{}]{Name: "rename"}
	_, err := Call(t.Context(), env.DataDir, rename, struct{}{})
	want := &ControlError{Code: "unknown_op", Message: `unknown op "rename"; node version test`}
	if got, ok := errors.AsType[*ControlError](err); !ok || *got != *want {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestOnlyStopIsServedBeforeReady(t *testing.T) {
	n := &Node{env: Env{Version: "test"}}
	_, err := n.dispatch("join")
	want := ControlError{Code: "not_ready", Message: "node is starting; it serves only stop until ready"}
	if got, ok := errors.AsType[*ControlError](err); !ok || *got != want {
		t.Fatalf("join before ready: err = %v, want %v", err, want)
	}
	if h, err := n.dispatch("stop"); h == nil || err != nil {
		t.Fatalf("stop before ready: handler %v, err %v", h != nil, err)
	}
}
