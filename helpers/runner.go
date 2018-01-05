package helpers

import (
	"context"
	"errors"
	"github.com/golang/glog"
	"io"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

const (
	FakeCommandExitOutput     = "exit"
	FakeCommandDiedOutput     = "died"
	FakeCommandCanceledOutput = "canceled"
)

var (
	FakeCommandExecutionError = errors.New("execution returned non-zero exit code")
)

// Runner represents something that can be run, and supports context semantics.
type Runner interface {
	Run(ctx context.Context) error
}

// CommandCreatorFunc creates runnable commands.
type CommandCreatorFunc func(stdout, stderr io.Writer, name string, args ...string) Runner

type commandRunner struct {
	cmd *exec.Cmd
}

// NewCommandRunner creates a new commandRunner that is ready to be run.
func NewCommandRunner(stdout, stderr io.Writer, name string, args ...string) Runner {
	// We can't use CommandContext here, because canceling execution only kills
	// the parent process, and can result in orphaned child processes.
	cmd := exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Set up a process group. This is needed, so that we can kill child
	// processes as well when we need to kill the parent process.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return &commandRunner{cmd}
}

// Run kicks off the command in a separate goroutine, and awaits completion through
// a channel. We also watch the context, and kill the process if execution is interrupted.
func (r *commandRunner) Run(ctx context.Context) error {
	// Kick it off.
	runCh := make(chan error)
	go func() {
		runCh <- r.cmd.Run()
	}()

	// Wait for execution to end, one way or the other.
	select {
	case err := <-runCh:
		// Execution ended normally; just pass the result along.
		return err
	case <-ctx.Done():
		// Execution is being canceled externally.
		// Kill the process through the process group, to handle children as well.
		if r.cmd.Process != nil {
			var err error
			var pgid int
			if pgid, err = syscall.Getpgid(r.cmd.Process.Pid); err == nil {
				if err = syscall.Kill(-pgid, syscall.SIGKILL); err == nil {
					return ctx.Err()
				}
			}

			// This is just for good measure.
			glog.Warningf("Failed to kill process via process group: %v; attempting to kill parent...", err)
			err = r.cmd.Process.Kill()
			if err != nil {
				glog.Warningf("Failed to kill process %d: %v.", r.cmd.Process.Pid, err)
			}
		}
		return ctx.Err()
	}
}

type fakeCommandRunner struct {
	stdout    io.Writer
	stderr    io.Writer
	timeout   time.Duration
	succeeded bool
}

// NewFakeCommandRunner creates a runner with controllable outcome. We can set a running
// time, and whether it exits successfully. It also writes predictable output to stdout/stderr,
// and respects context semantics, just like real command runners.
func NewFakeCommandRunner(stdout, stderr io.Writer, name string, args ...string) Runner {
	if len(args) != 2 {
		argsFatal(args...)
	}

	timeout, err := time.ParseDuration(args[0])
	if err != nil {
		argsFatal(args...)
	}

	succeeded, err := strconv.ParseBool(args[1])
	if err != nil {
		argsFatal(args...)
	}

	return &fakeCommandRunner{
		stdout:    stdout,
		stderr:    stderr,
		timeout:   timeout,
		succeeded: succeeded,
	}
}

func (c *fakeCommandRunner) Run(ctx context.Context) error {
	timer := time.NewTimer(c.timeout)

	select {
	case <-timer.C:
		if c.succeeded {
			c.stdout.Write([]byte(FakeCommandExitOutput))
			return nil
		}
		c.stderr.Write([]byte(FakeCommandDiedOutput))
		return FakeCommandExecutionError
	case <-ctx.Done():
		c.stderr.Write([]byte(FakeCommandCanceledOutput))
		return ctx.Err()
	}
}

func argsFatal(args ...string) {
	glog.Fatalln("args need to consist of timeout (duration) and succeeded (bool), but were: ", args)
}
