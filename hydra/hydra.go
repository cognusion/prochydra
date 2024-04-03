package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cognusion/go-sequence"
	"github.com/cognusion/prochydra/dictionary"
	"github.com/cognusion/prochydra/head"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Block of global vars, because lazy
var (
	// VERSION is the internal code revision number
	VERSION string = "0.1.0"
	// NUMCPU is the number of CPUs at starttime
	NUMCPU = runtime.NumCPU()
	// GOVERSION is the version of Go
	GOVERSION = runtime.Version()

	OutFormat = log.Ldate | log.Ltime | log.Lshortfile
	DebugOut  = log.New(io.Discard, "[DEBUG] ", 0)
	ErrorOut  = log.New(io.Discard, "[ERROR] ", 0)
	StdOut    = log.New(io.Discard, "", 0)
	StdErr    = log.New(io.Discard, "", 0)

	heads sync.Map                              // heads is a Map of all heads running that hydra is aware of
	idSeq = sequence.NewWithHashIDLength(0, 14) // idSeq is for IDs
	seq   = sequence.New(0)                     // seq is for heads to use in macros

	conf *viper.Viper
	dict dictionary.SimpleDict
)

// Rule: All errors in init() must be Fatal
func init() {

	pflag.Bool("debug", false, "Enable vociferous output")
	pflag.String("debuggoros", "", "Duration of wait between dumping goro stack if --debug, e.g. \"1s\" or \"100ms\"")

	pflag.String("exec", "", "Command to execute, if singular. Ignores many other options and should only be used for debugging")
	pflag.Bool("autorestart", false, "Enable autorestarts. Set --restartdelay to sleep in between")
	pflag.String("restartdelay", "0s", "Duration of wait between restarts, e.g. \"1s\" or \"100ms\"")

	pflag.String("log", "", "Path to file to log to, else stderr")
	pflag.String("outlog", "", "Path to file where stdout should log to, else stdout")
	pflag.String("errlog", "", "Path to file where stderr should log to, else stdout")
	pflag.Int("logsize", 100, "Maximum size, in MB, that the current log can be before rolling")
	pflag.Int("logbackups", 3, "Maximum number of rolled logs to keep")
	pflag.Int("logage", 28, "Maximum age, in days, to keep rolled logs")

	pflag.Int64("maxpss", 0, "Maximum PSS (in MB) each process is allowed before being killed")
	pflag.Int("seq", 0, "Integer to start a sequence at. {seq} in a command will be incremented per-command in a head instance (regardless of counts or mutations, starting at this number")
	pflag.Uint("uid", 0, "Run as uid (0 for current user)")
	pflag.Uint("gid", 0, "Run as gid (-uid must be set as well) (0 for current group)")

	pflag.Bool("version", false, fmt.Sprintf("Print the version (%s), and then exit", VERSION))
	pflag.Bool("dashc", false, "Wrap the commands in 'bash -c' instead of running them directly")
	config := pflag.String("config", "", "Config file to load")

	pflag.Parse()

	var err error
	conf, err = LoadConfig(*config)
	if err != nil {
		log.Fatalf("Error loading config '%s': %s\n", *config, err)
	}

	// Bind commandline flags to viper config
	conf.BindPFlags(pflag.CommandLine)

	// Short circuit init if --version
	if conf.GetBool("version") {
		return
	}

	// Early dictionary parsing.
	if macros := conf.GetStringMapString("macros"); len(macros) > 0 {
		dict = macros
	}

	// Set the ErrorOut
	ErrorOut = GetErrorLog(dict.Replacer(conf.GetString("log")), "[HEAD] ", OutFormat, conf.GetInt("logsize"), conf.GetInt("logbackups"), conf.GetInt("logage"))

	// Set the StdOut & StdErr logs
	StdOut = GetLog(dict.Replacer(conf.GetString("outlog")), "", 0, conf.GetInt("logsize"), conf.GetInt("logbackups"), conf.GetInt("logage"))
	StdErr = GetErrorLog(dict.Replacer(conf.GetString("errlog")), "", 0, conf.GetInt("logsize"), conf.GetInt("logbackups"), conf.GetInt("logage"))

	// Set the DebugOut, maybe
	if conf.GetBool("debug") {
		DebugOut = GetErrorLog(dict.Replacer(conf.GetString("log")), "[DEBUG] ", OutFormat, conf.GetInt("logsize"), conf.GetInt("logbackups"), conf.GetInt("logage"))
		conf.DebugTo(DebugOut.Writer()) // belch out the config debug output
	}

	// Init heads
	if headcheck := conf.Get("heads"); headcheck == nil {
		log.Fatalf("No heads detected in configuration. Aborting.\n")
	}

	// Sequence seeding, maybe
	if conf.GetInt("seq") > 0 {
		seq = sequence.New(conf.GetInt("seq"))
	}
}

func main() {

	if conf.GetBool("version") {
		fmt.Printf("Head %s\nGo   %s\nCPUs %d\n",
			VERSION,
			GOVERSION,
			NUMCPU)
		return
	}

	var (
		errorChan = make(chan error, 20)
		wg        sync.WaitGroup
	)

	if conf.GetBool("debug") && conf.GetDuration("debuggoros") != time.Duration(0) {
		// Fork off the stack dumper
		go func() {
			t := time.NewTicker(conf.GetDuration("debuggoros"))
			for {
				<-t.C
				buf := make([]byte, 1<<16)
				runtime.Stack(buf, true)
				numg := runtime.NumGoroutine()
				DebugOut.Printf("--->\n%s\nGoros: %d\n<---\n", buf, numg)
			}
		}()
	}

	// Fork off the error reader
	go func() {
		for e := range errorChan {
			if e != nil {
				ErrorOut.Println(e)
			}
		}
	}()

	// Fork off the INT/TERM signal handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		// Stop all heads
		heads.Range(func(k, v interface{}) bool {
			h := v.(*head.Head)
			if h != nil {
				DebugOut.Printf("Signalling head %s to stop...\n", k)
				h.Stop()
			}
			return true
		})
	}()

	// If we have an exec
	if exec := conf.GetString("exec"); exec != "" {
		rcommand := dict.Replacer(exec)

		// create the Head, and set some stuff
		var h *head.Head
		if conf.GetBool("dashc") {
			h = head.BashDashC(rcommand, errorChan)
		} else {
			lcommand, largs, err := CommandSplit(rcommand)
			if err != nil {
				ErrorOut.Fatalf("Error parsing command '%s': %s\n", rcommand, err)
			}
			h = head.New(lcommand, largs, errorChan)
		}

		// Set the head's properties from config
		h.DebugOut = DebugOut
		h.ErrOut = ErrorOut
		h.StdOut = StdOut
		h.Seq = seq
		h.MaxPSS = conf.GetInt64("maxpss")

		// Add this head to the list and waitgroup
		h.ID = idSeq.NextHashID()
		heads.Store(h.ID, h)
		wg.Add(1)

		// Run the head
		rs := h.Run()
		DebugOut.Printf("Conf: %s\n", h.String())
		DebugOut.Printf("Live: %s\n", rs)

		// Wait for this head to finish, or not
		go func(r *head.Head) {
			defer wg.Done()
			defer heads.Delete(r.ID)
			r.Wait()
		}(h)
	}
	//POST: A head from CLI may or may not be running

	// if we have HeadConfig objects
	if headcheck := conf.Get("heads"); headcheck != nil {
		confheads := make([]HeadConfig, len(headcheck.([]interface{})))
		conf.UnmarshalKey("heads", &confheads)

		// Iterate over the commands
		for _, hc := range confheads {

			rcommand := dict.Replacer(hc.Command)

			var h *head.Head

			// Create the Head
			if conf.GetBool("dashc") {
				h = head.BashDashC(rcommand, errorChan)
			} else {
				lcommand, largs, err := CommandSplit(rcommand)
				if err != nil {
					ErrorOut.Fatalf("Error parsing command '%s': %s\n", rcommand, err)
				}
				h = head.New(lcommand, largs, errorChan)
			}

			DebugOut.Printf("HeadC %s\n", hc.Command)

			// Set stuff
			h.DebugOut = DebugOut
			h.ErrOut = ErrorOut
			h.Seq = seq

			if hc.ChildEnvFile != "" {
				content, err := os.ReadFile(hc.ChildEnvFile)
				if err != nil {
					ErrorOut.Fatalf("Error reading %s: %v\n", hc.ChildEnvFile, err)
				}
				lines := strings.Split(string(content), "\n")
				DebugOut.Printf("\tHeadC ")
				h.SetChildEnv(lines)
			}

			// Setup the head, from config
			if hc.StdOutLog != "" {
				DebugOut.Printf("\tHeadC Custom StdOutLog: %s\n", hc.StdOutLog)
				h.StdOut = GetLog(dict.Replacer(hc.StdOutLog), "", 0, conf.GetInt("logsize"), conf.GetInt("logbackups"), conf.GetInt("logage"))
			} else {
				h.StdOut = StdOut
			}

			if hc.StdErrLog != "" {
				DebugOut.Printf("\tHeadC Custom StdErrLog: %s\n", hc.StdErrLog)
				h.StdErr = GetErrorLog(dict.Replacer(hc.StdErrLog), "", 0, conf.GetInt("logsize"), conf.GetInt("logbackups"), conf.GetInt("logage"))
			} else {
				h.StdErr = StdErr
			}

			if hc.Autorestart {
				DebugOut.Printf("\tHeadC Custom Autorestart: %t\n", hc.Autorestart)
				h.Autorestart(hc.Autorestart)
			} else {
				h.Autorestart(conf.GetBool("autorestart"))
			}

			if hc.RestartDelay > 0 {
				DebugOut.Printf("\tHeadC Custom Restartdelay: %s\n", hc.RestartDelay.String())
				h.RestartDelay = hc.RestartDelay
			} else {
				h.RestartDelay = conf.GetDuration("restartdelay")
			}

			if hc.MaxPSS > 0 {
				DebugOut.Printf("\tHeadC Custom MaxPSS: %d\n", hc.MaxPSS)
				h.MaxPSS = hc.MaxPSS
			} else {
				h.MaxPSS = conf.GetInt64("maxpss")
			}

			if hc.UID > 0 {
				DebugOut.Printf("\tHeadC Custom UID: %d\n", hc.UID)
				h.UID = hc.UID
			} else {
				h.UID = uint32(conf.GetInt("uid"))
			}

			if hc.GID > 0 {
				DebugOut.Printf("\tHeadC Custom GID: %d\n", hc.GID)
				h.GID = hc.GID
			} else {
				h.GID = uint32(conf.GetInt("gid"))
			}

			if hc.RestartsCritOver == nil {
				// Default -1 (off)
				h.Values.Store("RestartsCritOver", -1)
			} else {
				DebugOut.Printf("\tHeadC Custom RestartsCritOver: %v\n", hc.RestartsCritOver)
				h.Values.Store("RestartsCritOver", ValueSwitch(hc.RestartsCritOver))
			}

			if hc.RestartsWarnOver == nil {
				// Default -1 (off)
				h.Values.Store("RestartsWarnOver", -1)
			} else {
				DebugOut.Printf("\tHeadC Custom RestartsWarnOver: %v\n", hc.RestartsWarnOver)
				h.Values.Store("RestartsWarnOver", ValueSwitch(hc.RestartsWarnOver))
			}

			if hc.Timeout > 0 {
				DebugOut.Printf("\tHeadC Custom Timeout: %s\n", hc.Timeout.String())
				h.Timeout = hc.Timeout
			}

			if hc.Name != "" {
				DebugOut.Printf("\tHeadC Custom Name: %s\n", hc.Name)
				h.Values.Store("Name", hc.Name)
			}

			// bookkeeping
			h.ID = idSeq.NextHashID()
			heads.Store(h.ID, h)
			wg.Add(1)

			// Run the head
			rs := h.Run()
			DebugOut.Printf("Conf: %s\n", h.String())
			DebugOut.Printf("Live: %s\n", rs)

			// Wait for this head to finish, or not
			go func(r *head.Head) {
				defer wg.Done()
				defer heads.Delete(r.ID)
				r.Wait()
			}(h)
		}
	}

	// Wait for all the commands to end.
	wg.Wait()
}
