package node

import (
	"errors"
	"os"
	"testing"
	"time"
)

// restartable is env whose Restart starts the successor in this process
// and sends it on the returned channel.
func restartable(t *testing.T, env Env) (with Env, successor <-chan *Node) {
	t.Helper()
	next := make(chan *Node, 1)
	env.Restart = func() error {
		go func() {
			n, err := Start(t.Context(), env)
			if err != nil {
				t.Errorf("start the successor: %v", err)
				close(next)
				return
			}
			t.Cleanup(func() {
				n.Stop()
				<-n.Done()
			})
			next <- n
		}()
		return nil
	}
	return env, next
}

func TestRestartStartsASuccessorWithThePendingSettings(t *testing.T) {
	env, _ := testEnv(t)
	env, next := restartable(t, env)
	n := startTest(t, env)
	ports := "3"
	if _, _, err := SetConfig(t.Context(), env, []ConfigChange{{Key: "http.ports_to_try", Value: &ports}}); err != nil {
		t.Fatal(err)
	}
	if got := n.cfg.HTTP.PortsToTry; got != 10 {
		t.Fatalf("running http.ports_to_try = %d before the restart, want 10", got)
	}

	if err := n.Restart(); err != nil {
		t.Fatal(err)
	}
	if err := n.Restart(); !errors.Is(err, ErrAlreadyRestarting) {
		t.Fatalf("second restart = %v, want %v", err, ErrAlreadyRestarting)
	}
	waitDone(t, n)
	if n.Err() != nil {
		t.Fatalf("the restarted Node stopped with %v", n.Err())
	}
	var successor *Node
	select {
	case successor = <-next:
	case <-time.After(30 * time.Second):
		t.Fatal("no successor within 30s")
	}
	if successor == nil {
		t.FailNow()
	}
	status, err := successor.Status(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if status.PID != os.Getpid() || !status.StartedAt.After(n.started) || successor.cfg.HTTP.PortsToTry != 3 {
		t.Fatalf("successor PID %d started %s with http.ports_to_try %d; want PID %d, started after %s, 3",
			status.PID, status.StartedAt, successor.cfg.HTTP.PortsToTry, os.Getpid(), n.started)
	}
}

func TestAStoppingNodeRefusesARestart(t *testing.T) {
	env, _ := testEnv(t)
	env, next := restartable(t, env)
	n := startTest(t, env)
	n.Stop()
	if err := n.Restart(); !errors.Is(err, ErrStopping) {
		t.Fatalf("restart = %v, want %v", err, ErrStopping)
	}
	waitDone(t, n)
	select {
	case s := <-next:
		t.Fatalf("a stopped Node started a successor: %v", s)
	default:
	}
}

func TestANodeWithNoRestartRefusesOne(t *testing.T) {
	env, _ := testEnv(t)
	n := startTest(t, env)
	if err := n.Restart(); !errors.Is(err, ErrCannotRestart) {
		t.Fatalf("restart = %v, want %v", err, ErrCannotRestart)
	}
}

func TestRestartOfIsReadAndRemoved(t *testing.T) {
	environ, pid, err := restartOf([]string{"HOME=/home/a", "SLIPSTREAM_RESTART_OF=4242", "SLIPSTREAM_SEED=x"})
	if err != nil || pid != 4242 || len(environ) != 2 || environ[0] != "HOME=/home/a" || environ[1] != "SLIPSTREAM_SEED=x" {
		t.Fatalf("restartOf = %q, %d, %v", environ, pid, err)
	}
	if _, _, err := restartOf([]string{"SLIPSTREAM_RESTART_OF=x"}); err == nil || err.Error() != "SLIPSTREAM_RESTART_OF=x is not a PID" {
		t.Fatalf("err = %v", err)
	}
}
