package lerna

import (
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/cognusion/go-recyclable"
)

var rPool = recyclable.NewBufferPool()

// Run attempts to start a server listener to handle control requests. If error is returned,
// assume it failed. The returned chan can be closed to trigger a server shutdown.
func Run(protocol, socketAddress string, errorLog, debugLog *log.Logger) (chan struct{}, error) {

	if err := socketCleanup(socketAddress); err != nil {
		errorLog.Printf("Aborting. Error removing existing socket: %s\n", err)
		return nil, err
	}

	// start listener
	listener, err := net.Listen(protocol, socketAddress)
	if err != nil {
		errorLog.Printf("Aborting. Error attempting to listen to socket: %s\n", err)
		return nil, err
	}

	// create quitChan to return, and handle cleanup in a goro
	quit := make(chan struct{})
	go func(l net.Listener, sa string) {
		<-quit
		debugLog.Println("runServer stop signalled")
		l.Close()
		socketCleanup(sa)
	}(listener, socketAddress)

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
			go echo(conn, debugLog)
		}
	}(listener)

	// All set. Return the quitChan.
	return quit, nil
}

func echo(conn net.Conn, outLog *log.Logger) {
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

	buf.Reset([]byte(strings.ToUpper(buf.String()))) // uppercase the buffer

	outLog.Printf("server Writing...")
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
