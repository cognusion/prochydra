// Package athena is a system to track the PSS memory usage of an os.Process
// and kill it if the usage exceeds the stated Limit. Limits may be cancelled and
// new Limits estabilished.
package athena

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/cognusion/go-humanity"
)

// MemoryGuard is our encapsulating mechanation, and should only be acquired via a New helper
type MemoryGuard struct {
	// Name is a name to use in lieu of PID for messaging
	Name string
	// Interval is a time.Duration to wait between checking usage
	Interval time.Duration
	// DebugOut is a logger for debug information
	DebugOut *log.Logger
	// ErrOut is a logger for StdErr coming from a process
	ErrOut *log.Logger
	// KillChan will be closed if/when the process is killed
	KillChan chan struct{}

	cancelled      chan bool
	nokill         bool // true if the process should not be killed in overmemory cases
	proc           *os.Process
	lastPss        int64
	statsFrequency time.Duration
}

// NewMemoryGuard takes an os.Process and returns a MemoryGuard for that process
func NewMemoryGuard(Process *os.Process) *MemoryGuard {
	return &MemoryGuard{
		proc:           Process,
		Interval:       1 * time.Second,
		KillChan:       make(chan struct{}),
		cancelled:      make(chan bool, 1),
		DebugOut:       log.New(io.Discard, "", 0),
		ErrOut:         log.New(io.Discard, "", 0),
		statsFrequency: time.Minute,
	}
}

// SetNoKill changes the default behavior of the MemoryGuard, to not kill the
// process if it exceeds the specified limit.
func (m *MemoryGuard) SetNoKill() {
	m.nokill = true
}

// StatsFrequency updates the internal frequency to which statistics are emitted to
// the debug logger. Default is 1 minute.
func (m *MemoryGuard) StatsFrequency(freq time.Duration) {
	m.statsFrequency = freq
}

// PSS returns the last known PSS value for the watched process,
// or the current value, if there was no last value
func (m *MemoryGuard) PSS() int64 {
	if lp := atomic.LoadInt64(&m.lastPss); lp > 0 {
		return lp
	}
	pss, err := getPss(m.proc.Pid)
	if err != nil {
		return 0
	}
	return pss
}

// Cancel stops any Limit() operations. After calling Cancel this
// MemoryGuard will be non-functional
func (m *MemoryGuard) Cancel() {
	select {
	case m.cancelled <- true:
		// cancelling
	default:
		// already cancelled
	}
}

// Limit takes the max usage (in Bytes) for the process.
// and acts on the PSS of the process uness UseRSS is true
func (m *MemoryGuard) Limit(max int64) {

	go func() {
		var name string
		if m.Name != "" {
			name = m.Name
		} else {
			name = fmt.Sprintf("%d", m.proc.Pid)
		}

		since := time.Now()
		for {
			select {
			case <-m.cancelled:
				m.DebugOut.Printf("[%s] MemoryGuard Cancelled!\n", name)
				return
			default:
			}

			var (
				xss int64
				err error
			)

			xss, err = getPss(m.proc.Pid)
			if err != nil {
				m.ErrOut.Printf("[%s] MemoryGuard getPss Error: %s\n", name, err)
				time.Sleep(m.Interval)
				continue
			} else {
				atomic.StoreInt64(&m.lastPss, xss)
			}

			if xss > max {
				m.ErrOut.Printf("[%s] MemoryGuard ALERT! %s Limit %s\n", name, humanity.ByteFormat(xss), humanity.ByteFormat(max))
				close(m.KillChan)
				if m.nokill {
					// don't kill it
				} else {
					// kill it
					m.proc.Kill()
				}
				return
			} else if time.Since(since) >= m.statsFrequency {
				// Belch out the stats every so often
				since = time.Now()
				m.DebugOut.Printf("[%s] MemoryGuard: %s Limit %s\n", name, humanity.ByteFormat(xss), humanity.ByteFormat(max))
			}

			time.Sleep(m.Interval)
		}
	}()
}

// getPss takes a pid, and returns the sum of PSS page sizes in Bytes, or an error
func getPss(pid int) (int64, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/smaps", pid))
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var (
		res int64
		pfx = []byte("Pss:")
	)

	r := bufio.NewScanner(f)
	for r.Scan() {
		line := r.Bytes()
		if bytes.HasPrefix(line, pfx) {
			var size int64
			_, err := fmt.Sscanf(string(line[4:]), "%d", &size)
			if err != nil {
				return 0, err
			}
			res += size
		}
	}
	if err := r.Err(); err != nil {
		return 0, err
	}

	return res * 1024, nil
}
