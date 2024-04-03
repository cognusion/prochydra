package athena

import (
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func ExampleMemoryGuard() {
	// Get a handle on our process
	us, _ := os.FindProcess(os.Getpid())

	// Create a new MemoryGuard around the process
	mg := NewMemoryGuard(us)
	mg.Limit(512 * 1024 * 1024) // Set the HWM memory limit. You can change this at any time

	// Do stuff that is memory-hungry

	// Stop guarding. After this, if you want to guard the process,
	// you need to NewMemoryGuard() again
	mg.Cancel()
}

func Test_MemoryGuardOnUsPSSRapid(t *testing.T) {
	Convey("When a MemoryGuard is running on us", t, func() {
		us, _ := os.FindProcess(os.Getpid())
		mg := NewMemoryGuard(us)
		mg.Limit(400 * 1024 * 1024) // we won't actually hit this, right?
		defer mg.Cancel()

		Convey("and we spam the PSS() function, we don't get killed, and a PSS is returned", func() {
			for c := 0; c < 1000; c++ {
				So(mg.PSS(), ShouldBeGreaterThan, 0)
			}
		})
	})
}

func Test_MemoryGuardOnUsPSS(t *testing.T) {
	Convey("When a MemoryGuard is running on us", t, func() {
		us, _ := os.FindProcess(os.Getpid())
		mg := NewMemoryGuard(us)
		mg.Name = "bob"
		mg.Limit(400 * 1024 * 1024) // we won't actually hit this, right?
		defer mg.Cancel()

		Convey("we don't get killed, and a PSS is returned", func() {
			So(mg.PSS(), ShouldBeGreaterThan, 0)
		})
	})
}

func Test_MemoryGuardOnUsDelay(t *testing.T) {
	Convey("When a MemoryGuard is running on us", t, func() {
		us, _ := os.FindProcess(os.Getpid())
		mg := NewMemoryGuard(us)
		mg.StatsFrequency(time.Second)
		mg.Limit(400 * 1024 * 1024) // we won't actually hit this, right?
		defer mg.Cancel()

		Convey("After 2 seconds", func() {
			time.Sleep(2 * time.Second)

			Convey("we don't get killed, and a PSS is returned", func() {
				So(mg.PSS(), ShouldBeGreaterThan, 0)
			})
		})
	})
}

func Test_MemoryGuardSmapsBadPid(t *testing.T) {
	Convey("When a MemoryGuard checks SMAPS for an invalid pid", t, func() {
		_, e := getPss(-10)
		Convey("it returns an error", func() {
			So(e, ShouldNotBeNil)
		})
	})
}

func Test_MemoryGuardGetPss(t *testing.T) {
	Convey("When a MemoryGuard checks SMAPS with a valid pid for PSS", t, func() {
		_, e := getPss(os.Getpid())
		Convey("it doesn't return an error", func() {
			So(e, ShouldBeNil)
		})
	})
}

func Test_MemoryGuardGetPssBadPid(t *testing.T) {
	Convey("When a MemoryGuard is running on us", t, func() {
		us, _ := os.FindProcess(os.Getpid())
		mg := NewMemoryGuard(us)
		mg.proc.Pid = -10
		mg.Limit(400 * 1024 * 1024) // we won't actually hit this, right?
		defer mg.Cancel()

		Convey("and we have a bad pid, we don't get killed, and a PSS of 0 is returned", func() {
			So(mg.PSS(), ShouldEqual, 0)
		})
	})
}

func Test_MemoryGuardCancelSpamPSS(t *testing.T) {
	Convey("When a MemoryGuard is running on us", t, func() {
		us, _ := os.FindProcess(os.Getpid())
		mg := NewMemoryGuard(us)
		mg.Limit(400 * 1024 * 1024) // we won't actually hit this, right?
		defer mg.Cancel()

		Convey("and we spam the cancel function, we don't get killed or blocked", func() {
			for c := 0; c < 1000; c++ {
				mg.Cancel()
			}
			So(true, ShouldBeTrue)
		})
	})
}

func Test_MemoryGuardKillPSS(t *testing.T) {
	Convey("When a MemoryGuard is running on us", t, func() {
		us, _ := os.FindProcess(os.Getpid())
		mg := NewMemoryGuard(us)
		mg.Interval = time.Second
		mg.SetNoKill()

		Convey("and set a really low threshold, we'll get killed", func() {
			defer mg.Cancel()
			mg.Limit(1024) // 1KB

			<-mg.KillChan // wait for the kill
			So(true, ShouldBeTrue)
		})
	})
}
