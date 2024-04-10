package head

import (
	"github.com/cognusion/go-sequence"
	"github.com/cognusion/go-slippycounter"
	"github.com/cognusion/prochydra/athena"
	"github.com/cognusion/randomnames"

	sq "github.com/Hellseher/go-shellquote"

	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Head is a struct to contain a process a run. You can Run() the same Head multiple times if
// you need clones.
type Head struct {
	// ID is a generated ID based on the sequence
	ID string
	// RestartDelay specified the duration to wait between restarts
	RestartDelay time.Duration
	// MaxPSS specifies the maximum PSS size a process may have before being killed
	MaxPSS int64
	// DebugOut is a logger for debug information
	DebugOut *log.Logger
	// ErrOut is a logger for errors, generally, emitted by a Head
	ErrOut *log.Logger
	// StdErr is a logger for StdErr coming from a process
	StdErr *log.Logger
	// StdOut is a logger for StdOut coming from a process
	StdOut *log.Logger
	// UID is the uid to run as (leave unset for current user)
	UID uint32
	// GID is the gid to run as (leave unset for current group)
	GID uint32
	// Seq is a pointer to an initialized sequencer
	Seq *sequence.Seq
	// Values is a map for implementors to store key-value pairs. Never consulted by the Head library.
	Values sync.Map
	// Timeout is a duration after which the process running is stopped, subject to  Autorestart
	Timeout time.Duration
	// StdInNoNL is a boolean to describe if a NewLine should *not* be appended to lines written to StdIn.
	// This is advisory-only, and respected by hydra but not necessarily others.
	StdInNoNL bool

	wg           sync.WaitGroup
	restarts     uint64
	errors       uint64
	errorChan    chan<- error
	rawErrorChan chan error
	command      string
	args         []string
	ctx          context.Context
	cancel       context.CancelFunc
	restartsMin  *slippycounter.SlippyCounter
	mgInterval   time.Duration
	status       atomic.Value
	autoRestart  atomic.Value
	stdIn        io.WriteCloser
	stdInLock    sync.Mutex
	childEnv     []string
}

// BashDashC creates a head that handles the command in its entirety running as a "bash -c command"
func BashDashC(command string, errorChan chan error) *Head {
	return New("bash", []string{"-c", command}, errorChan)
}

// New returns a Head struct, ready to Run() the command with the arguments
// specified. It would be wise to ensure the errorChan reader is ready before calling
// Run() to prevent goro plaque.
func New(command string, args []string, errorChan chan error) *Head {

	ctx, cancel := context.WithCancel(context.Background())
	r := Head{
		command:      command,
		args:         args,
		errorChan:    errorChan,
		rawErrorChan: errorChan,
		restarts:     0,
		ctx:          ctx,
		cancel:       cancel,
		DebugOut:     log.New(io.Discard, "", 0),
		ErrOut:       log.New(io.Discard, "", 0),
		StdOut:       log.New(io.Discard, "", 0),
		StdErr:       log.New(io.Discard, "", 0),
		restartsMin:  slippycounter.NewSlippyCounter(1 * time.Minute),
		mgInterval:   30 * time.Second,
	}
	r.status.Store("init")
	r.autoRestart.Store(false)
	return &r
}

// Clone returns a new Head intialized the same as the current
func (r *Head) Clone() *Head {
	c := New(r.command, r.args, r.rawErrorChan)

	c.RestartDelay = r.RestartDelay
	c.MaxPSS = r.MaxPSS
	c.DebugOut = r.DebugOut
	c.ErrOut = r.ErrOut
	c.StdErr = r.StdErr
	c.StdOut = r.StdOut
	c.UID = r.UID
	c.GID = r.GID
	c.Seq = r.Seq
	c.Values = *copyValues(&r.Values)
	c.Timeout = r.Timeout
	c.StdInNoNL = r.StdInNoNL

	return c
}

// Write implements io.Writer to ease writing to StdIn
func (r *Head) Write(p []byte) (n int, err error) {
	r.stdInLock.Lock()
	defer r.stdInLock.Unlock()
	return r.stdIn.Write(p)
}

// SetChildEnv takes a list of key=value strings to pass to all spawned processes
func (r *Head) SetChildEnv(env []string) {
	r.childEnv = env
}

// Autorestart sets whether or not we will automatically restart Heads that "complete"
func (r *Head) Autorestart(doit bool) {
	r.autoRestart.Store(doit)
}

// SetMgInterval sets the interval at which the memory usage is checked, for use with MaxPSS.
// Default 30s.
func (r *Head) SetMgInterval(interval time.Duration) {
	r.mgInterval = interval
}

// String returns the original command and arguments in a line
func (r *Head) String() string {
	return fmt.Sprintf("%s %s", r.command, sq.Join(r.args...))
}

// Run executes a subprocess of the command and arguments specified, restarting
// it if applicable. The returned channel returns the name of the once it is
// running, or closes it without a value if it will not run.
func (r *Head) Run() string {

	if r.status.Load() == "running" {
		return ""
	}

	r.wg.Add(1)
	r.status.Store("running")

	stringChan := make(chan string, 1)
	procname := randomnames.SafeRandomAdjectiveAnimal()

	go func(name string, s chan<- string) {
		defer r.wg.Done()
		defer r.status.Store("done")

		// Make the [short]name macro
		re := regexp.MustCompile(`\W`)
		shortname := re.ReplaceAllString(strings.ToLower(name), "")

		lcommand := r.command

		largs := make([]string, len(r.args))
		copy(largs, r.args)

		for i, arg := range r.args {

			// Replace all instances of the name macro globally
			arg = strings.Replace(arg, "{name}", shortname, -1)

			// Iterate over each instance of {seq} so we replace sequentially
			if r.Seq != nil {
				for strings.Contains(arg, "{seq}") {
					arg = strings.Replace(arg, "{seq}", r.Seq.NextHashID(), 1)
				}
			}

			// Put arg back in place
			largs[i] = arg

		}

		// Send the macro-expanded command string back to the caller
		s <- fmt.Sprintf("%s %s", lcommand, sq.Join(largs...))

		r.DebugOut.Printf("%s/%s Starting (%s)...", name, r.ID, shortname)
		defer r.DebugOut.Printf("%s/%s Exiting (%s)...", name, r.ID, shortname)

		for {
			var (
				mg      *athena.MemoryGuard
				cmd     *exec.Cmd
				lcancel = func() {}
				lctx    = r.ctx
			)

			if r.Timeout > 0 {
				// Timeout, remap the context
				lctx, lcancel = context.WithTimeout(r.ctx, r.Timeout)
			}

			cmd = exec.CommandContext(lctx, lcommand, largs...)

			if r.UID > 0 {
				// Run as
				cmd.SysProcAttr = &syscall.SysProcAttr{}
				cmd.SysProcAttr.Credential = &syscall.Credential{Uid: r.UID, Gid: r.GID}
				r.DebugOut.Printf("\t%+v\n", cmd.SysProcAttr.Credential)
			}

			if r.childEnv != nil {
				r.DebugOut.Printf("Setting Env %v\n", r.childEnv)
				cmd.Env = r.childEnv
			}

			// grab stderr and stdout and stdin
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				r.errorHandler(fmt.Errorf("%s/%s: 'stdoutpipe' %w", name, r.ID, err))
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				r.errorHandler(fmt.Errorf("%s/%s: 'stderrpipe' %w", name, r.ID, err))

			}
			stdIn, err := cmd.StdinPipe()
			if err != nil {
				r.errorHandler(fmt.Errorf("%s/%s: 'stdinpipe' %w", name, r.ID, err))
			}
			// Because races are real:
			r.stdInLock.Lock()
			r.stdIn = stdIn
			r.stdInLock.Unlock()

			// Copy the output to the logs.
			go ReadLogger(stdout, r.StdOut, r.errorChan)
			go ReadLogger(stderr, r.StdErr, r.errorChan)

			// Go go gadget command!
			if err := cmd.Start(); err != nil {
				r.errorHandler(fmt.Errorf("%s/%s: 'starting' %w", name, r.ID, err))
				lcancel()
			} else {
				// We're running!

				// Set up memory guard
				if r.MaxPSS > 0 {
					mg = athena.NewMemoryGuard(cmd.Process)
					mg.Name = name
					mg.DebugOut = r.DebugOut
					mg.ErrOut = r.ErrOut
					mg.Interval = r.mgInterval
					mg.Limit(r.MaxPSS * 1024 * 1024)
				}

				// Wait until the cmd is done
				if err := cmd.Wait(); err != nil {
					select {
					case <-r.ctx.Done():
					// Context has been cancelled, don't send errors

					case <-lctx.Done():
						// Local context cancelled, might matter
						if lctx.Err() != nil {
							r.errorHandler(fmt.Errorf("%s/%s: 'local' %w", name, r.ID, lctx.Err()))
						}

					default:
						// Global context is clear
						r.errorHandler(fmt.Errorf("%s/%s: 'waiting' %w", name, r.ID, err))
					}
				}

				// If there's a local context, cancel it
				lcancel()

				if r.MaxPSS > 0 {
					mg.Cancel()
				}
			}

			if r.autoRestart.Load() == false {
				// We done.
				return
			}

			select {
			case <-r.ctx.Done():
				// Context cancelled, stop signalled
				r.DebugOut.Printf("%s/%s Cancelling...", name, r.ID)
				return
			default:
			}

			// else do it again.. after a nap, maybe
			time.Sleep(r.RestartDelay)
			atomic.AddUint64(&r.restarts, 1)
			r.restartsMin.Add(1)
			r.DebugOut.Printf("%s/%s Restarting...", name, r.ID)
		}
	}(procname, stringChan)

	s := <-stringChan
	return s
}

// Stop signals all of the running processes to die. May generate error
// output thereafter.
func (r *Head) Stop() {
	r.DebugOut.Println("Stop signalled")
	r.autoRestart.Store(false) // prevent more restarts
	r.cancel()                 // Cancel the global context
	r.restartsMin.Close()      // Close the slippy counter
	r.DebugOut.Println("Stop completed")
}

// Errors returns the current number of errors sent to the error chan
func (r *Head) Errors() uint64 {
	return atomic.LoadUint64(&r.errors)
}

// Restarts returns the current number of restarts for this Head instance
func (r *Head) Restarts() uint64 {
	return atomic.LoadUint64(&r.restarts)
}

// RestartsPerMinute returns the number of restarts for this Head instance in
// the last minute
func (r *Head) RestartsPerMinute() uint64 {
	return uint64(r.restartsMin.Count())
}

// Wait blocks until the Run() process has completed
func (r *Head) Wait() {
	r.wg.Wait()
}

// errorHandler is an internal regurgitator to asynchronously
// spew errors down the errorChan
func (r *Head) errorHandler(e error) {
	atomic.AddUint64(&r.errors, 1)

	// Fork off a goro to eventually submit the error maybe
	go func(err error) {
		r.DebugOut.Printf("Pre-channel error: '%s'\n", e)

		defer func() {
			// In poor form, it is possible we send on a closed channel, which
			// immutably causes a panic. We need to close that channel to prevent
			// leaks, and we've moved on so the result is garbage.
			rec := recover()
			if rec != nil {
				r.DebugOut.Printf("errorHandler Panic: %v\n", rec)
			}
		}()

		timeout := time.After(time.Second)
		select {
		case <-timeout:
			// We're discarding error output, if nothing is around to
			// read it after a second.
			r.DebugOut.Printf("errorChan timed out, dropping error: %s\n", e)
			return
		case r.errorChan <- err:
			// yay
			r.DebugOut.Printf("error submitted: %s\n", e)
		}
	}(e)
}

// copyValues iterates over the passed map any copies the values to a new map,
// returning a pointer to it. Used by Clone().
func copyValues(values *sync.Map) *sync.Map {
	var cp sync.Map
	values.Range(func(k, v interface{}) bool {
		cp.Store(k, v)
		return true
	})

	return &cp
}
