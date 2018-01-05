package e2etest

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/glog"
)

// fakeCommand-based dispatcher.
func newDispatcher() Dispatcher {
	return &runnerDispatcher{helpers.NewFakeCommandRunner}
}

// fakeCommand-based dispatcherSystemRunner.
func newSystemRunner() SystemRunner {
	return &dispatcherSystemRunner{
		dispatcher: newDispatcher(),
	}
}

// Since this is a unit test, we'll treat "one second" as "forever".
func blockingContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 1*time.Second)
}

func fakeCommandDescription(Label string, args ...string) CommandDescription {
	return CommandDescription{
		Label:  Label,
		Name:   "",
		Args:   args,
		Stdout: bytes.NewBufferString(""),
		Stderr: bytes.NewBufferString(""),
	}
}

// readExitChannel reads a single CmdExit from the channel, while
// applying a timeout to ensure we don't block forever if we've broken something.
func readExitChannel(exitCh chan CmdExit) (CmdExit, error) {
	ctx, _ := blockingContext()
	select {
	case result := <-exitCh:
		return result, nil
	case <-ctx.Done():
		return CmdExit{}, ctx.Err()
	}
}

func getBufferString(w io.Writer) string {
	buf, ok := w.(*bytes.Buffer)
	if !ok {
		glog.Fatalf("Writer is of type %T, but should be *bytes.Buffer.", w)
	}

	return buf.String()
}

func checkCommandOutput(cmd CommandDescription, outWant, errWant string, t *testing.T) {
	outGot := getBufferString(cmd.Stdout)
	if outGot != outWant {
		t.Errorf("wanted stdout %s, got %s", outWant, outGot)
	}
	errGot := getBufferString(cmd.Stderr)
	if errGot != errWant {
		t.Errorf("wanted stderr %s, got %s", errWant, errGot)
	}
}

// Due to the concurrency nature of this setup, we need a way to wait for processes
// to complete before proceeding with the next step. Since fake commands always
// output to their stdout/stderr, we just wait for something to show up.
//
// In practice, this is just to ensure commands that end after zero seconds are allowed
// to finish running before we invoke 'cancel' in the test, so this isn't expected to
// block longer than 10ms.
func fakeCommandWait(cmd CommandDescription) {
	t := time.NewTicker(10 * time.Millisecond)
	ctx, _ := blockingContext()
	for {
		select {
		case <-t.C:
			if len(getBufferString(cmd.Stderr))+len(getBufferString(cmd.Stdout)) > 0 {
				t.Stop()
				return
			}
		case <-ctx.Done():
			glog.Fatal("Command that was expected to terminate never did.")
		}
	}
}

func TestRunnerDispatcher_Success(t *testing.T) {
	exitCh := make(chan CmdExit)
	ctx, _ := blockingContext()
	cmd := fakeCommandDescription("cmd", "0", "true")
	newDispatcher().Dispatch(ctx, cmd, exitCh)

	want := CmdExit{"cmd", nil}
	got, err := readExitChannel(exitCh)
	if err != nil {
		t.Errorf("reading from channel unexpectedly timed out")
	}

	if want != got {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestRunnerDispatcher_Failure(t *testing.T) {
	exitCh := make(chan CmdExit)
	ctx, _ := blockingContext()
	cmd := fakeCommandDescription("cmd", "0", "false")
	newDispatcher().Dispatch(ctx, cmd, exitCh)

	got, err := readExitChannel(exitCh)
	if err != nil {
		t.Errorf("reading from channel unexpectedly timed out")
	}

	if got.Label != "cmd" {
		t.Errorf("wanted label cmd, but got %s", got.Label)
	}

	if got.Err != helpers.FakeCommandExecutionError {
		t.Errorf("wanted error %v, got %v", helpers.FakeCommandExecutionError, got.Err)
	}
}

func TestRunnerDispatcher_ContextCancel(t *testing.T) {
	exitCh := make(chan CmdExit)
	ctx, cancel := blockingContext()
	cmd := fakeCommandDescription("cmd", "1s", "true")

	cancel()
	newDispatcher().Dispatch(ctx, cmd, exitCh)

	got, err := readExitChannel(exitCh)
	if err != nil {
		t.Errorf("reading from channel unexpectedly timed out")
	}

	if got.Label != "cmd" {
		t.Errorf("wanted label cmd, but got %s", got.Label)
	}

	if got.Err != ctx.Err() {
		t.Errorf("wanted error %v, got %v", helpers.FakeCommandExecutionError, got.Err)
	}
}

func TestRunSystem_DuplicateLabel(t *testing.T) {
	ctx, cancel := blockingContext()
	cmd := fakeCommandDescription("cmd", "1s", "true")
	runner := newSystemRunner()
	err := runner.Start(ctx, cancel, []CommandDescription{cmd, cmd})
	if err == nil {
		t.Error("wanted error, got no error")
	}
	runner.Stop()
}

func TestRunSystem_DoubleStart(t *testing.T) {
	ctx, cancel := blockingContext()
	cmd := fakeCommandDescription("cmd", "1s", "true")
	runner := newSystemRunner()
	err := runner.Start(ctx, cancel, []CommandDescription{cmd})
	if err != nil {
		t.Errorf("wanted no error, got error %v", err)
	}
	err = runner.Start(ctx, cancel, []CommandDescription{cmd})
	if err == nil {
		t.Error("wanted error, got no error")
	}
	runner.Stop()
}

func TestRunSystem_ManualCancel(t *testing.T) {
	ctx, cancel := blockingContext()
	cmds := []CommandDescription{
		fakeCommandDescription("process-1", "1s", "true"),
		fakeCommandDescription("process-2", "1s", "true"),
	}

	runner := newSystemRunner()

	cancel()
	err := runner.Start(ctx, cancel, cmds)
	if err != nil {
		t.Errorf("wanted no error, got %v", err)
	}

	runner.Stop()

	checkCommandOutput(cmds[0], "", helpers.FakeCommandCanceledOutput, t)
	checkCommandOutput(cmds[1], "", helpers.FakeCommandCanceledOutput, t)
}

func TestRunSystem_ProcessDies(t *testing.T) {
	ctx, cancel := blockingContext()
	cmds := []CommandDescription{
		fakeCommandDescription("process-1", "1s", "true"),
		fakeCommandDescription("process-2", "0", "false"),
	}

	runner := newSystemRunner()

	err := runner.Start(ctx, cancel, cmds)
	if err != nil {
		t.Errorf("wanted no error, got %v", err)
	}

	fakeCommandWait(cmds[1])
	runner.Stop()

	checkCommandOutput(cmds[0], "", helpers.FakeCommandCanceledOutput, t)
	checkCommandOutput(cmds[1], "", helpers.FakeCommandDiedOutput, t)
}

func TestRunSystem_ProcessExits(t *testing.T) {
	ctx, cancel := blockingContext()
	cmds := []CommandDescription{
		fakeCommandDescription("process-1", "1s", "true"),
		fakeCommandDescription("process-2", "0", "true"),
	}

	runner := newSystemRunner()

	err := runner.Start(ctx, cancel, cmds)
	if err != nil {
		t.Errorf("wanted no error, got %v", err)
	}

	fakeCommandWait(cmds[1])
	runner.Stop()

	checkCommandOutput(cmds[0], "", helpers.FakeCommandCanceledOutput, t)
	checkCommandOutput(cmds[1], helpers.FakeCommandExitOutput, "", t)
}
