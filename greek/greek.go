package greek

import (
	"io"
)

// Verbs
const (
	List    = Verb("list")
	Stop    = Verb("stop")
	Exec    = Verb("exec")
	Send    = Verb("send")
	NilVerb = Verb("")
)

// Nouns
const (
	Heads   = Noun("heads")
	Head    = Noun("head")
	NilNoun = Noun("")
)

type (
	// Verb is a language construct to declare what should be done
	Verb = string
	// Noun is a language construct to declare what we doing something to
	Noun = string
)

// Request is a structure to request information or action from the server.
type Request struct {
	Verb    Verb
	Noun    Noun
	Data    string
	Waiting bool
	Chan    chan Response
}

// Response is a structure to respond if a Request expects it.
type Response struct {
	// If IsFinal, no more responses will be forthcoming
	IsFinal bool
	// Data contains the body of the response, and should be Close()d after reading.
	Data io.ReadCloser
	// if Error != nil, Data is undefined.
	Error error
}

// ToVerb returns the Verb of the string, or NilVerb.
func ToVerb(s string) Verb {
	switch s {
	case "list":
		return List
	case "stop":
		return Stop
	case "send":
		return Send
	case "exec":
		return Exec
	default:
		return NilVerb
	}
}

// ToNoun returns the Noun of the string, or NilNoun.
func ToNoun(s string) Noun {
	switch s {
	case "heads":
		return Heads
	case "head":
		return Head
	default:
		return NilNoun
	}
}
