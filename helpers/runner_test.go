// +build !windows

package helpers

import (
	"bytes"
	"context"
	"testing"
	"time"
)

// Since this is a unit test, we'll treat "one second" as "forever".
func blockingContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 1*time.Second)
}

// TODO: This test won't cross-compile on Windows. We should check
//       runtime.GOOS and branch accordingly.
func TestNewCommand_Success(t *testing.T) {
	stderr := bytes.NewBufferString("")
	stdout := bytes.NewBufferString("")
	ctx, _ := blockingContext()
	want := "woohoo!"
	runner := NewCommandRunner(stdout, stderr, "echo", "-n", want)
	err := runner.Run(ctx)
	if err != nil {
		t.Errorf("wanted no error, got %v", err)
	}

	if want != stdout.String() {
		t.Errorf("wanted output %s, but got %s", want, stdout.String())
	}
}

func TestNewCommand_Error(t *testing.T) {
	stderr := bytes.NewBufferString("")
	stdout := bytes.NewBufferString("")
	ctx, _ := blockingContext()
	runner := NewCommandRunner(stdout, stderr, "false")
	err := runner.Run(ctx)
	if err == nil {
		t.Error("wanted error, but got no error")
	}
}

func TestFakeCommand_Success(t *testing.T) {
	stderr := bytes.NewBufferString("")
	stdout := bytes.NewBufferString("")
	ctx, _ := blockingContext()
	runner := NewFakeCommandRunner(stdout, stderr, "", "0", "true")
	err := runner.Run(ctx)

	if err != nil {
		t.Errorf("wanted no error, but got %v", err)
	}

	if stdout.String() != FakeCommandExitOutput {
		t.Errorf("wanted stdout %s, got %s", FakeCommandExitOutput, stdout.String())
	}
	if stderr.String() != "" {
		t.Errorf("wanted empty stderr, got %s", stderr.String())
	}
}

func TestFakeCommand_Failure(t *testing.T) {
	stderr := bytes.NewBufferString("")
	stdout := bytes.NewBufferString("")
	ctx, _ := blockingContext()
	runner := NewFakeCommandRunner(stdout, stderr, "", "0", "false")
	err := runner.Run(ctx)

	if err == nil {
		t.Error("wanted error, got no error")
	}

	if err != FakeCommandExecutionError {
		t.Errorf("wanted %v, but got %v", FakeCommandExecutionError, ctx.Err())
	}

	if stdout.String() != "" {
		t.Errorf("wanted empty stdout, got %s", stdout.String())
	}
	if stderr.String() != FakeCommandDiedOutput {
		t.Errorf("wanted stderr %s, got %s", FakeCommandDiedOutput, stderr.String())
	}
}

func TestFakeCommand_ContextCancel(t *testing.T) {
	stderr := bytes.NewBufferString("")
	stdout := bytes.NewBufferString("")
	ctx, cancel := blockingContext()
	runner := NewFakeCommandRunner(stdout, stderr, "", "1s", "true")

	cancel()
	err := runner.Run(ctx)

	if err == nil {
		t.Error("wanted error, got no error")
	}

	if err != context.Canceled {
		t.Errorf("wanted %v, but got %v", context.Canceled, err)
	}

	if stdout.String() != "" {
		t.Errorf("wanted empty stdout, got %s", stdout.String())
	}
	if stderr.String() != FakeCommandCanceledOutput {
		t.Errorf("wanted stderr %s, got %s", FakeCommandCanceledOutput, stderr.String())
	}
}
