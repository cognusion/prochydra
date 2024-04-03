package main

import (
	"fmt"

	sq "github.com/Hellseher/go-shellquote"
)

// Strings is a weakly-named construct to help make dealing with commands and arguments easier
type Strings struct {
	commands []string
	args     [][]string
}

// String returns the stringified version of the commands
func (s *Strings) String() string {
	return fmt.Sprintf("%s", s.commands)
}

// Get returns the index-refrerenced command, with any arguments, or an error if the index is invalid
func (s *Strings) Get(i int) (string, []string, error) {
	if i < len(s.commands) {
		return s.commands[i], s.args[i], nil
	}
	return "", []string{}, fmt.Errorf("invalid item index: %d", i)
}

// Count returns the number of commands
func (s *Strings) Count() int {
	return len(s.commands)
}

// Set splits the passed value into and command and its args, and appends them, or returns an error
func (s *Strings) Set(value string) error {

	command, args, err := CommandSplit(value)
	if err != nil {
		return err
	}

	s.commands = append(s.commands, command)
	s.args = append(s.args, args)
	return nil
}

// Type returns the ... type
func (s *Strings) Type() string {
	return "Strings"
}

// CommandSplit is a helper to split a command string into the command and a list of arguments, or an error
func CommandSplit(command string) (string, []string, error) {
	args, err := sq.Split(command)
	if err != nil {
		return "", []string{}, err
	} else if len(args) < 1 {
		// So they specified a command, but it was all spaces. WTF?
		return "", []string{}, fmt.Errorf("command is all spaces")
	}

	return args[0], args[1:], nil
}
