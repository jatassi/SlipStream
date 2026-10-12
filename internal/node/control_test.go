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
	select {
	case <-n.Done():
	default:
		t.Fatal("StopNode returned before the Node was done")
	}
	if n.Err() != nil {
		t.Fatalf("node stopped with %v", n.Err())
	}
	startTest(t, env)
}

func TestUnknownOpNamesTheNodeVersion(t *testing.T) {
	env, _ := testEnv(t)
	startTest(t, env)
	join := op.Op[struct{}, struct{}]{Name: "join"}
	_, err := Call(t.Context(), env.DataDir, join, struct{}{})
	want := &ControlError{Code: "unknown_op", Message: `unknown op "join"; node version test`}
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
