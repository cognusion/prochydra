package main

import (
	"fmt"

	sq "github.com/Hellseher/go-shellquote"
)

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
