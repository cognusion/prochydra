package main

import (
	"fmt"
	"io"

	sq "github.com/Hellseher/go-shellquote"
	"github.com/cognusion/go-recyclable"
	"github.com/cognusion/prochydra/greek"
	"github.com/cognusion/prochydra/head"
)

var (
	rPool = recyclable.NewBufferPool() // Recyclable buffers for server IPC use
)

func handleRequest(req *greek.Request) {
	// Short circuit bad requests
	if req.Noun == greek.NilNoun || req.Verb == greek.NilVerb {
		if req.Waiting {
			req.Chan <- greek.Response{
				IsFinal: true,
				Error:   fmt.Errorf("improper Noun or Verb sent"),
			}
		}
		return
	}

	var buf *recyclable.Buffer
	if req.Waiting {
		// We will need a buffer, so get one ready.
		buf = rPool.Get()
		buf.Reset([]byte{})
	}

	switch req.Noun {
	case greek.Head:
		// Specific Head.

		// Ensure there is at least something in the Data field
		if req.Data == "" {
			if req.Waiting {
				buf.Close()
				req.Chan <- greek.Response{
					IsFinal: true,
					Error:   fmt.Errorf("head operation requested but no head specified"),
				}
			}
			return
		}

		// Split the data field
		rd, err := sq.Split(req.Data)
		if err != nil {
			if req.Waiting {
				buf.Close()
				req.Chan <- greek.Response{
					IsFinal: true,
					Error:   fmt.Errorf("error while splitting data: %s", err),
				}
			}
			return
		}

		// Retrieve the Head
		v, ok := heads.Load(rd[0])
		if !ok {
			if req.Waiting {
				buf.Close()
				req.Chan <- greek.Response{
					IsFinal: true,
					Error:   fmt.Errorf("requested Head ID does not exist"),
				}
			}
			return
		}
		// We should have a valid head now!
		h := v.(*head.Head)
		data := sq.Join(rd[1:]...)

		switch req.Verb {
		case greek.Stop:
			// Stop specific Head
			h.Stop()
			if req.Waiting {
				io.WriteString(buf, "Head Stopped\n")
				req.Chan <- greek.Response{
					IsFinal: true,
					Data:    buf,
				}
			}
			return
		case greek.Send:
			// Send to specific Head
			io.WriteString(h, data+"\n")
			// TODO: grab stdout?
			if req.Waiting {
				io.WriteString(buf, "Head Written\n")
				req.Chan <- greek.Response{
					IsFinal: true,
					Data:    buf,
				}
			}

			return
		}
	case greek.Heads:
		switch req.Verb {
		case greek.Stop:
			// Stop Heads
			heads.Range(func(k, v interface{}) bool {
				h := v.(*head.Head)
				if h != nil {
					h.Stop()
				}
				return true
			})
			if req.Waiting {
				io.WriteString(buf, "All Heads Stopped\n")
				req.Chan <- greek.Response{
					IsFinal: true,
					Data:    buf,
				}
			}
			return
		case greek.List:
			// LIST HEADS
			if req.Waiting {
				heads.Range(func(k, v interface{}) bool {
					h := v.(*head.Head)
					if h != nil {
						io.WriteString(buf, fmt.Sprintf("%s: %s - %d\n", h.ID, h.String(), h.Restarts()))
					}
					return true
				})

				req.Chan <- greek.Response{
					IsFinal: true,
					Data:    buf,
				}
			}
			return
		}
	}

	if req.Waiting {
		buf.Close()
		req.Chan <- greek.Response{
			IsFinal: true,
			Error:   fmt.Errorf("improper request sent"),
		}
	}
}
