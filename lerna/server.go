package lerna

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	sq "github.com/Hellseher/go-shellquote"
	"github.com/cognusion/go-recyclable"
	"github.com/cognusion/prochydra/greek"
)

var rPool = recyclable.NewBufferPool()

// Run attempts to start a server listener to handle control requests. If error is returned,
// assume it failed. The returned chan can be closed to trigger a server shutdown.
func Run(protocol, socketAddress string, errorLog, debugLog *log.Logger) (chan struct{}, <-chan greek.Request, error) {

	if err := socketCleanup(socketAddress); err != nil {
		errorLog.Printf("Aborting. Error removing existing socket: %s\n", err)
		return nil, nil, err
	}

	// start listener
	listener, err := net.Listen(protocol, socketAddress)
	if err != nil {
		errorLog.Printf("Aborting. Error attempting to listen to socket: %s\n", err)
		return nil, nil, err
	}

	// create quitChan to return, and handle cleanup in a goro
	quitChan := make(chan struct{})
	go func(l net.Listener, sa string) {
		<-quitChan
		debugLog.Println("runServer stop signalled")
		l.Close()
		socketCleanup(sa)
	}(listener, socketAddress)

	// create requestChan to send requests to hydra
	requestChan := make(chan greek.Request, 1)
	go func(l net.Listener) {
		debugLog.Println("server launched...")
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				errorLog.Printf("Aborting. Listener cannot accept: %s\n", err)
				return
			}

			debugLog.Println(">>> accepted")
			go handleConnection(conn, requestChan, debugLog)
		}
	}(listener)

	// All set. Return the quitChan.
	return quitChan, requestChan, nil
}

func handleConnection(conn net.Conn, requestChan chan<- greek.Request, outLog *log.Logger) {
	defer conn.Close()
	outLog.Printf("Connected: %s\n", conn.RemoteAddr().Network())

	buf := rPool.Get()
	defer buf.Close()
	buf.Reset([]byte{})
	outLog.Println("server Reading...")
	_, err := io.Copy(buf, conn)
	if err != nil {
		outLog.Println(err)
		return
	}

	outLog.Println("\t:", buf.String())
	args, err := sq.Split(buf.String())
	if err != nil {
		outLog.Printf("Request was an error! %v\n", err)
		io.WriteString(conn, fmt.Sprintf("An error parsing request: %s\n", err))
		return
	} else if len(args) < 2 {
		outLog.Printf("Request is improper!\n")
		io.WriteString(conn, "Request is improper\n")
		return
	}

	// Make nouns and verbs lowercase
	args[0] = strings.ToLower(args[0])
	args[1] = strings.ToLower(args[1])

	respChan := make(chan greek.Response)
	requestChan <- greek.Request{
		Verb:    greek.ToVerb(args[0]),
		Noun:    greek.ToNoun(args[1]),
		Data:    sq.Join(args[2:]...),
		Waiting: true,
		Chan:    respChan,
	}

	outLog.Println("server Waiting for response from control...")
	resp := <-respChan
	if resp.IsFinal {
		close(respChan)
	}
	if resp.Error != nil {
		outLog.Printf("Response was an error! %v\n", resp.Error)
		io.WriteString(conn, fmt.Sprintf("An error was returned: %s\n", resp.Error))
		return
	}
	defer resp.Data.Close() // make sure that gets closed

	// Reuse our existing buffer
	buf.ResetFromReader(resp.Data)

	outLog.Println("server Writing...")
	_, err = io.Copy(conn, buf)
	if err != nil {
		outLog.Println(err)
		return
	}

	buf.Seek(0, 0)
	outLog.Println("<<< ", buf.String())
}

func socketCleanup(socketAddress string) error {
	if _, serr := os.Stat(socketAddress); serr == nil {
		if rerr := os.RemoveAll(socketAddress); rerr != nil {
			return rerr
		}
	}
	return nil
}
