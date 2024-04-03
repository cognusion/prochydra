package main

import (
	"github.com/spf13/viper"

	"fmt"
	"os"
	"strings"
	"time"
)

// HeadConfig is a configuration struct for object-based configs
type HeadConfig struct {
	// Name is a shorter name used if available in outputs, so the full command string doesn't have to be output
	Name string
	// Command is the full execution command to run
	Command string
	// Number is the count of instances of Command to execute
	Number int
	// Autorestart is whether to rerun Command if it exits
	Autorestart bool
	// StdOutLog is where to redirect captured stdout
	StdOutLog string
	// StdErrLog is where to redirect captures stderr
	StdErrLog string
	// RestartDelay specified the duration to wait between restarts
	RestartDelay time.Duration
	// MaxPSS specifies the maximum PSS size a process may have before being killed
	MaxPSS int64
	// UID is the uid to run as
	UID uint32
	// GID is the gid to run as
	GID uint32
	// RestartsWarnOver sets the HWM for cumulative restarts, before a Warning is set in the healthcheck. Default nil (off)
	RestartsWarnOver interface{}
	// RestartsCritOver sets the HWM for cumulative restarts, before a Critical is set in the healthcheck. Default nil (off)
	RestartsCritOver interface{}
	// RPMWarnOver sets the HWM for restarts-per-minute, before a Warning is set in the healthcheck. Default 0
	RPMWarnOver interface{}
	// RPMCritOver sets the HWM for restarts-per-minute, before a Critical is set in the healthcheck. Default 2
	RPMCritOver interface{}
	// Timeout is a duration after which the process running is stopped, subject to  Autorestart
	Timeout time.Duration
	// ChildEnvFile is a file of key=value pairs, one per line, that create the environment for the child processes
	// if unset, the parent environment will be inherhited
	ChildEnvFile string
}

// ValueSwitch returns -1 if the v is nil or not an int, otherwise returns the int value
func ValueSwitch(v interface{}) int {
	if v == nil {
		return -1
	}
	switch v := v.(type) {
	case int:
		return v
	default:
		return -1
	}
}

// ValueBomb returns the uint64 value of v and true IFF ok is true and ValueSwitch(v) is positive
// The odd syntax is to accommodate the direct output of a sync.Map.Load() operation
func ValueBomb(v interface{}, ok bool) (uint64, bool) {
	if !ok {
		return 0, false
	}

	if vi := ValueSwitch(v); vi >= 0 {
		return uint64(vi), true
	}
	return 0, false
}

// LoadConfig creates a new config, loads any environment config, and then any config from the specified file
func LoadConfig(configFilename string) (*viper.Viper, error) {
	v := viper.New()

	v.AutomaticEnv()
	v.SetEnvPrefix("head")

	if configFilename != "" {
		configFilenames := strings.Split(configFilename, ",")
		v.SetConfigFile(configFilenames[0])

		err := v.ReadInConfig()
		if err != nil {
			if _, ok := err.(viper.ConfigParseError); ok {
				return nil, err
			}
			return nil, fmt.Errorf("unable to locate Config file '%s'.(%s)", configFilenames[0], err)
		}
		for _, configFile := range configFilenames[1:] {
			file, err := os.Open(configFile) // For read access.
			if err != nil {
				return nil, fmt.Errorf("unable to open config file '%s': %s", configFile, err)
			}
			defer file.Close()
			if err = v.MergeConfig(file); err != nil {
				return nil, fmt.Errorf("unable to parse/merge Config file '%s': %s", configFile, err)
			}
		}
	}

	err := loadDefaults(v)
	if err != nil {
		return v, err
	}

	return v, nil
}

func loadDefaults(v *viper.Viper) error {

	v.SetDefault("debug", false)                 // Enable vociferous output
	v.SetDefault("debuggoros", time.Duration(0)) // Duration of wait between dumping goro stack if --debug, e.g. "1s" or "100ms" (0 to disable)
	v.SetDefault("log", "")                      // Path to file to log to, else stderr
	v.SetDefault("logsize", 100)                 // Maximum size, in MB, that the currently log can be before rolling
	v.SetDefault("logbackups", 3)                // Maximum number of rolled logs to keep
	v.SetDefault("logage", 28)                   // Maximum age, in days, to keep rolled logs

	// Container for head definitions
	v.SetDefault("heads", make([]interface{}, 0))

	// These globals also impact per-head defaults if unset
	v.SetDefault("outlog", "")                     // Path to file where stdout should log to, else stdout
	v.SetDefault("errlog", "")                     // Path to file where stderr should log to, else stderr
	v.SetDefault("command", []string{})            // Command(s) to run. Can be specified multiple times
	v.SetDefault("num", []int{1})                  // Number of copies of the process to run. MUST either be set exactly once, or the same number of times as --command is called, and in the desired order of such (Default 1)
	v.SetDefault("uid", uint(0))                   // Run as uid (0 for current user)
	v.SetDefault("gid", uint(0))                   // Run as gid (-uid must be set as well) (0 for current group)
	v.SetDefault("maxpss", int64(0))               // Maximum PSS (in MB) each process is allowed before being killed
	v.SetDefault("autorestart", false)             // Enable autorestarts. Set --restartdelay to sleep in between
	v.SetDefault("restartdelay", time.Duration(0)) // Duration of wait between restarts, e.g. "1s" or "100ms" (0 for no delay)

	return nil
}
