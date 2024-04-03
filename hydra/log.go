package main

import (
	"io"
	"log"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// GetLog gets a standard-type log
func GetLog(filename, prefix string, format, size, backups, age int) *log.Logger {
	return getLog(filename, prefix, format, size, backups, age, os.Stdout)
}

// GetErrorLog gets an error-type log
func GetErrorLog(filename, prefix string, format, size, backups, age int) *log.Logger {
	return getLog(filename, prefix, format, size, backups, age, os.Stderr)
}

// getLog abstracts all the things
func getLog(filename, prefix string, format, size, backups, age int, defaultWriter io.Writer) (l *log.Logger) {
	if filename == "" {
		// Nothing provided, use the defaults
		l = log.New(defaultWriter, prefix, format)
	} else {
		// File, use lumberjack
		l = log.New(&lumberjack.Logger{
			Filename:   filename,
			MaxSize:    size, // megabytes
			MaxBackups: backups,
			MaxAge:     age, // days
		}, prefix, format)
	}
	return
}
