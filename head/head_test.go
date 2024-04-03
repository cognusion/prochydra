package head

import (
	"fmt"
	"io"
	"sync"

	"github.com/cognusion/go-sequence"
	"github.com/fortytw2/leaktest"
	sq "github.com/Hellseher/go-shellquote"
	. "github.com/smartystreets/goconvey/convey"

	"context"
	"errors"
	"log"
	"testing"
	"time"
)

func Test_HeadRunOnlyOnce(t *testing.T) {
	defer leaktest.Check(t)()

	errorChan := make(chan error, 1)
	//defer close(errorChan)
	Convey("When a Head Initializes, and Run is called more than once, subsequent calls obviously bail", t, func() {
		r := New("sleep", []string{"30"}, errorChan)
		defer r.Stop()

		// First run is good
		name := r.Run()
		So(name, ShouldNotBeZeroValue)

		// Hit it 10 more times, and no more are running.
		for range 10 {
			iname := r.Run()
			So(iname, ShouldBeZeroValue)
		}
	})
}

func Test_HeadClone(t *testing.T) {
	defer leaktest.Check(t)()

	errorChan := make(chan error, 1)
	//defer close(errorChan)
	Convey("When a Head Initializes, and Run is called, and then there is a clone, and run is called more than once on the clone, subsequent calls obviously bail", t, func() {
		r := New("sleep", []string{"30"}, errorChan)
		defer r.Stop()

		// First run is good
		name := r.Run()
		So(name, ShouldNotBeZeroValue)

		cr := r.Clone()
		defer cr.Stop()

		// First run is good
		cname := cr.Run()
		So(cname, ShouldNotBeZeroValue)

		// Hit it 10 more times, and no more are running.
		for range 10 {
			iname := cr.Run()
			So(iname, ShouldBeZeroValue)
		}
	})
}

func Test_HeadStop(t *testing.T) {
	defer leaktest.Check(t)()

	errorChan := make(chan error, 1)
	//defer close(errorChan)
	Convey("When a Head Initializes that will run for 30 seconds", t, func() {
		r := New("sleep", []string{"30"}, errorChan)
		defer r.Stop()

		Convey("and a Stop is issued.. it should end in under a second", func() {
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			start := time.Now()
			r.Stop()
			r.Wait()
			diff := time.Since(start)
			So(diff, ShouldBeLessThan, 1*time.Second)
		})

	})
}

func Test_HeadTimeout(t *testing.T) {

	errorChan := make(chan error, 1)
	defer close(errorChan)
	Convey("When a Head Initializes with a timeout", t, func() {
		r := New("/usr/bin/sleep", []string{"3"}, errorChan)
		defer r.Stop()

		r.Timeout = 100 * time.Millisecond
		Convey("the timeout should be called", func() {
			start := time.Now()
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			r.Wait()

			So(time.Now().UnixNano(), ShouldBeLessThan, start.Add(1*time.Second).UnixNano())
			e := <-errorChan
			So(e, ShouldNotBeNil)
			So(errors.Is(e, context.DeadlineExceeded), ShouldBeTrue)
		})

	})
}

func Test_HeadTimeoutClear(t *testing.T) {

	errorChan := make(chan error, 2)
	defer close(errorChan)
	Convey("When a Head Initializes with a timeout, but is faster than the timeout", t, func() {
		r := New("echo", []string{"hello"}, errorChan)
		defer r.Stop()

		r.Timeout = 1 * time.Second
		Convey("the timeout should not be called", func() {
			start := time.Now()
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			r.Wait()

			So(time.Now().UnixNano(), ShouldBeLessThan, start.Add(r.Timeout).UnixNano())

			errorChan <- nil // errorChan should be empty, thus block on a pop.
			e := <-errorChan
			So(e, ShouldBeNil)

		})

	})
}

func Test_HeadAutorestart(t *testing.T) {

	errorChan := make(chan error, 1)
	//defer close(errorChan)
	Convey("When a Head Initializes that autorestarts", t, func() {
		r := New("echo hi", nil, errorChan)
		defer r.Stop()
		r.Autorestart(true)

		Convey("the restart counts should be > 0", func() {
			time.AfterFunc(500*time.Millisecond, func() { r.Stop() })
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			r.Wait()
			So(r.Restarts(), ShouldBeGreaterThan, 0)
			So(r.RestartsPerMinute(), ShouldBeGreaterThan, 0)
		})

	})
}

func Test_HeadAutorestartCancel(t *testing.T) {

	errorChan := make(chan error, 1)
	//defer close(errorChan)
	Convey("When a Head Initializes that autorestarts", t, func() {
		r := New("echo hi", nil, errorChan)
		defer r.Stop()
		r.Autorestart(true)

		Convey("the restart counts should be > 0", func() {
			time.AfterFunc(500*time.Millisecond, func() { r.cancel() })
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			r.Wait()
			So(r.Restarts(), ShouldBeGreaterThan, 0)
			So(r.RestartsPerMinute(), ShouldBeGreaterThan, 0)
		})

	})
}

func Test_HeadString(t *testing.T) {
	errorChan := make(chan error, 1)
	//defer close(errorChan)
	Convey("When a Head Initializes", t, func() {
		r := New("echo hi", nil, errorChan)
		defer r.Stop()

		Convey("and the String() is inspected", func() {
			So(r.String(), ShouldEqual, "echo hi"+" ")
		})

	})
}

func Test_HeadName(t *testing.T) {

	errorChan := make(chan error, 1)
	//defer close(errorChan)
	Convey("When a Head Initializes", t, func() {
		r := New("echo hi", []string{"{name}"}, errorChan)
		defer r.Stop()

		Convey("and Run, the name macro should be expanded", func() {
			s := r.Run()
			So(s, ShouldNotBeZeroValue)
			So(s, ShouldNotEqual, "echo hi"+" {name}")
		})

	})
}

func Test_HeadUidGid(t *testing.T) {

	errorChan := make(chan error, 1)
	//defer close(errorChan)

	Convey("When a Head Initializes, and configured with uid/gid 99", t, func() {
		r := New("echo hi", nil, errorChan)
		defer r.Stop()

		r.UID = uint32(99)
		r.GID = uint32(99)

		Convey("and Run, the name macro should be expanded", func() {
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
		})

	})

}

func Test_HeadSequenceFail(t *testing.T) {

	errorChan := make(chan error, 1)
	//defer close(errorChan)

	Convey("When a Head Initializes", t, func() {
		r := New("echo hi", []string{"{seq}"}, errorChan)
		defer r.Stop()

		Convey("and Run, the sequence should fail", func() {
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			So(name, ShouldNotEqual, "echo hi"+" 0")
		})

	})
}

func Test_HeadSequenceWin(t *testing.T) {

	var seq sequence.Seq
	errorChan := make(chan error, 1)
	//defer close(errorChan)

	Convey("When a Head Initializes", t, func() {
		r := New("echo hi", []string{"{seq}"}, errorChan)
		defer r.Stop()
		r.Seq = &seq

		Convey("and Run, the sequence should win!", func() {
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			So(name, ShouldEqual, "echo hi"+" 1")
		})

	})
}

func Test_SequenceSet(t *testing.T) {
	var seq sequence.Seq
	errorChan := make(chan error, 1)
	//defer close(errorChan)

	r := New("echo hi", nil, errorChan)
	r.Seq = &seq
	defer r.Stop()

	Convey("When a Sequence is initialized and Set to 42", t, func() {
		r.Seq = sequence.New(42)
		Convey("the value of Get should be 43... 44 .. 45", func() {
			So(r.Seq.Next(), ShouldEqual, 43)
			So(r.Seq.Next(), ShouldEqual, 44)
			So(r.Seq.Next(), ShouldEqual, 45)
		})
	})
}

func Test_SequenceZOneTwo(t *testing.T) {
	var seq sequence.Seq
	errorChan := make(chan error, 1)
	//defer close(errorChan)

	r := New("echo hi", nil, errorChan)
	r.Seq = &seq
	defer r.Stop()

	Convey("When a Sequence is initialized", t, func() {
		Convey("the value of Get should be one .. two .. three", func() {
			So(r.Seq.Next(), ShouldEqual, 1)
			So(r.Seq.Next(), ShouldEqual, 2)
			So(r.Seq.Next(), ShouldEqual, 3)
		})
	})
}

func Test_HeadMaxPSS(t *testing.T) {
	Convey("When a command runs", t, func() {
		echan := make(chan error, 1)
		r := New("tests/mem.sh", []string{}, echan)
		defer r.Stop()
		r.MaxPSS = 1 // 1 MB max
		r.mgInterval = 3 * time.Second
		Convey("and memory grows above mss", func() {
			start := time.Now()
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			<-echan // wait for the kill message
			stop := time.Now()

			Convey("it is killed in about 3s, and an error is registers", func() {
				So(stop.Sub(start), ShouldBeLessThanOrEqualTo, 5*time.Second)
				So(r.Errors(), ShouldBeGreaterThan, 0)
			})

		})

	})
}

func Test_HeadStdOutTrap(t *testing.T) {
	defer leaktest.Check(t)()

	errorChan := make(chan error, 1)
	var buf Sbuffer

	Convey("When a Head Initializes", t, func() {
		r := New("echo", []string{"hello", "world"}, errorChan)
		defer r.Stop()
		r.StdOut = log.New(&buf, "", 0)

		Convey("and Runs, the buffer is correct", func() {
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			r.Wait()

			So(buf.String(), ShouldEqual, "hello world\n")
		})
	})
}

func Test_HeadStdInStdOutTrap(t *testing.T) {
	defer leaktest.Check(t)()

	errorChan := make(chan error, 1)
	var buf Sbuffer

	Convey("When a Head Initializes with a 'cat'", t, func() {
		r := New("cat", []string{}, errorChan)
		defer r.Stop()
		r.StdOut = log.New(&buf, "", 0)

		Convey("and Runs, with 'hello world' being sent over stdin, the buffer is correct", func() {
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			time.Sleep(time.Second)
			io.WriteString(r, "hello world\n")
			time.Sleep(time.Second)
			So(buf.String(), ShouldEqual, "hello world\n")
		})
	})
}

func Test_HeadDashCStdOutTrap(t *testing.T) {
	defer leaktest.Check(t)()

	errorChan := make(chan error, 1)
	var buf Sbuffer

	Convey("When a Head (DashC) Initializes", t, func() {
		r := BashDashC("echo hello world", errorChan)
		defer r.Stop()
		r.StdOut = log.New(&buf, "", 0)

		Convey("and Runs, the buffer is correct", func() {
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			r.Wait()

			So(buf.String(), ShouldEqual, "hello world\n")
		})
	})
}

func Test_HeadDashCStdOutTrapPipe(t *testing.T) {
	defer leaktest.Check(t)()

	errorChan := make(chan error, 1)
	var buf Sbuffer

	Convey("When a Head (DashC) Initializes", t, func() {
		r := BashDashC("echo hello world | cat", errorChan)
		defer r.Stop()
		r.StdOut = log.New(&buf, "", 0)

		Convey("and Runs, the buffer is correct", func() {
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			r.Wait()

			So(buf.String(), ShouldEqual, "hello world\n")
		})
	})
}

func Test_HeadStdErrTrap(t *testing.T) {
	defer leaktest.Check(t)()

	errorChan := make(chan error, 1)
	var buf Sbuffer

	args, _ := sq.Split("bash -c 'echo gbye world 1>&2'")
	command := args[0]
	args = args[1:]

	Convey("When a Head Initializes", t, func() {
		r := New(command, args, errorChan)
		defer r.Stop()
		r.StdErr = log.New(&buf, "", 0)

		Convey("and Runs, the buffer is correct", func() {
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			r.Wait()

			So(buf.String(), ShouldEqual, "gbye world\n")
		})

	})
}

func Test_HeadChildEnv(t *testing.T) {
	defer leaktest.Check(t)()

	errorChan := make(chan error, 1)
	var buf Sbuffer

	Convey("When a Head Initializes, and the ChildEnv is set", t, func() {
		r := New("tests/vartest.sh", []string{}, errorChan)
		r.SetChildEnv([]string{"THEVAR=WORLD"})
		defer r.Stop()
		r.StdOut = log.New(&buf, "", 0)

		Convey("and Runs, the buffer is correct", func() {
			name := r.Run()
			So(name, ShouldNotBeZeroValue)
			r.Wait()

			So(buf.String(), ShouldEqual, "hello WORLD\n")
		})
	})
}

func Test_CopyValues(t *testing.T) {
	Convey("When a sync.Map is created, and copyValues is called on it, the resulting sync.Map is identical.", t, func() {
		var m1 sync.Map

		m1.Store("name", "bob")
		m1.Store("age", 42)
		m1.Store("score", 6.12345)
		m1.Store("Error", fmt.Errorf("an error"))
		m1.Store("Nil", nil)

		m2 := copyValues(&m1)

		name, ok := m2.Load("name")
		So(ok, ShouldBeTrue)
		So(name, ShouldEqual, "bob")

		age, ok := m2.Load("age")
		So(ok, ShouldBeTrue)
		So(age, ShouldEqual, 42)

		score, ok := m2.Load("score")
		So(ok, ShouldBeTrue)
		So(score, ShouldEqual, 6.12345)

		err, ok := m2.Load("Error")
		So(ok, ShouldBeTrue)
		So(err, ShouldBeError)

		nill, ok := m2.Load("Nil")
		So(ok, ShouldBeTrue)
		So(nill, ShouldBeNil)

	})
}
