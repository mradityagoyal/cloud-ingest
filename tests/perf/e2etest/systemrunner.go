// +build !windows

package e2etest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/glog"
)

var (
	alreadyRunningErr = errors.New("already running")
)

// CommandDescription describes a command that can be run. Label is a unique
// user-provided handle, for management purposes.
type CommandDescription struct {
	Label, Name    string
	Args           []string
	Stdout, Stderr io.Writer
}

// CmdExit is what a command passes back through the result channel after exiting.
// Err is nil for any successful execution (completes without interruption with exit-code = 0).
type CmdExit struct {
	Label string
	Err   error
}

// Dispatcher dispatches runnable commands, and is primarily meant to be used with dispatcherSystemRunner.
// That said, it is safe to use it as a stand-alone.
type Dispatcher interface {
	Dispatch(ctx context.Context, cmd CommandDescription, exitCh chan CmdExit)
}

type runnerDispatcher struct {
	cmdCreator helpers.CommandCreatorFunc
}

// NewRunnerDispatcher creates a dispatcher that uses the command runner.
func NewRunnerDispatcher() Dispatcher {
	return &runnerDispatcher{helpers.NewCommandRunner}
}

// Dispatch creates a runner from the command description, and runs it in a goroutine.
// When the runner completes, the error code, along with its label, are passed back to
// whatever is monitoring them.
func (d *runnerDispatcher) Dispatch(ctx context.Context, cmd CommandDescription, exitCh chan CmdExit) {
	runner := d.cmdCreator(cmd.Stdout, cmd.Stderr, cmd.Name, cmd.Args...)

	go func() {
		exitCh <- CmdExit{
			Label: cmd.Label,
			Err:   runner.Run(ctx),
		}
	}()
}

// Internal type, only useful to dispatcherSystemRunner.
type processTracker map[string]bool

func (pt processTracker) addProcess(label string) error {
	if _, ok := pt[label]; ok {
		return fmt.Errorf("duplicate label %s detected", label)
	}

	pt[label] = true
	return nil
}

func (pt processTracker) tombstoneProcess(label string) error {
	if _, ok := pt[label]; !ok {
		return fmt.Errorf("process %s does not exist", label)
	}

	pt[label] = false
	return nil
}

func (pt processTracker) isDone() bool {
	for _, v := range pt {
		if v {
			return false
		}
	}
	return true
}

// SystemRunner runs a system of multiple components (commands). These components are not expected
// to terminate on their own, unless they crash. If that happens, then everything is brought down.
//
// The CancelFunc that is passed to Start is expected to affect ctx. We don't just create a child,
// because we want a call to cancel to affect other components relying on this context as well. As
// this component evolves, it may be best to change this behavior and take in a quit channel instead.
type SystemRunner interface {
	Start(ctx context.Context, cancel context.CancelFunc, commands []CommandDescription) error
	Stop()
}

type dispatcherSystemRunner struct {
	dispatcher Dispatcher
	started    struct {
		sync.Mutex
		val bool
	}
	cancel         context.CancelFunc
	monitorCh      chan CmdExit
	processTracker processTracker
	wg             sync.WaitGroup
}

// NewDispatcherSystemRunner creates a dispatcher-based SystemRunner.
func NewDispatcherSystemRunner() SystemRunner {
	return &dispatcherSystemRunner{
		dispatcher: NewRunnerDispatcher(),
	}
}

// Start kicks off a number of processes in parallel.
//
// These processes are never expected to terminate.
//
// ctx must be cancelable, with the cancel function passed in. We don't just create a new
// cancelable context from it, because we want to easily signal to the outside that everything
// is stopping.
func (tr *dispatcherSystemRunner) Start(ctx context.Context, cancel context.CancelFunc, commands []CommandDescription) error {
	tr.started.Lock()
	defer tr.started.Unlock()
	if tr.started.val {
		return alreadyRunningErr
	}

	// Kick off monitoring, so we can respond to failures immediately.
	// Defer to ensure it happens even if we return an error during setup,
	// to avoid bad states.
	defer func() {
		tr.wg.Add(1)
		go func() {
			defer tr.wg.Done()
			tr.monitorForTermination()
		}()
	}()

	// Set this right away. If it fails while setting up, it still needs to be stopped.
	tr.started.val = true
	tr.cancel = cancel
	tr.processTracker = make(processTracker)
	tr.monitorCh = make(chan CmdExit)

	// Add commands.
	for _, c := range commands {
		glog.Infof("Starting %s...", c.Label)
		err := tr.processTracker.addProcess(c.Label)
		if err != nil {
			return err
		}
		tr.dispatcher.Dispatch(ctx, c, tr.monitorCh)
	}

	return nil
}

// monitorForTermination is an internal function that watches for signals coming in.
// This is a blocking function and is run in a separate goroutine.
func (tr *dispatcherSystemRunner) monitorForTermination() {
	for !tr.processTracker.isDone() {
		msg := <-tr.monitorCh

		if msg.Err == context.Canceled {
			glog.Infof("Process %s has exited.", msg.Label)
		} else {
			// If this process terminated on its own, cancel the rest.
			if msg.Err == nil {
				glog.Infof("Process %s has exited.", msg.Label)
			} else {
				glog.Warningf("Process %s has died, with error %v; Issuing cancel.", msg.Label, msg.Err)
			}
			tr.cancel()
		}

		// Mark process as exited. There is no action we can take on this
		// error, other than log it.
		err := tr.processTracker.tombstoneProcess(msg.Label)
		if err != nil {
			glog.Errorf("Failed to tombstone process %s: %v.", msg.Label, err)
		}
	}
}

// Stops whatever is running. This is a blocking operation. We call the cancel function,
// and wait for things to clear out.
func (tr *dispatcherSystemRunner) Stop() {
	tr.started.Lock()
	defer tr.started.Unlock()
	if !tr.started.val {
		return
	}

	tr.cancel()
	tr.wg.Wait()
	close(tr.monitorCh)
	tr.started.val = false
}
