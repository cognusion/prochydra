package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"

	sq "github.com/Hellseher/go-shellquote"
)

func main() {

	c, err := net.Dial("unix", "/tmp/hydra.sock")
	if err != nil {
		fmt.Printf("Error dialing: %s\n", err)
		return
	}
	defer c.Close()

	fmt.Println("Writing...")
	if err = write(c, sq.Join(os.Args[1:]...)); err != nil {
		fmt.Printf("Error writing: %s\n", err)
		return
	}

	fmt.Println("Reading...")
	buf := bytes.Buffer{}
	_, err = buf.ReadFrom(c)
	if err != nil {
		fmt.Printf("Error reading from Conn: %s\n", err)
		return
	}
	fmt.Println(buf.String())

}

func write(c net.Conn, message string) error {
	if cw, ok := c.(interface{ CloseWrite() error }); ok {
		defer cw.CloseWrite()
	} else {
		return fmt.Errorf("connection doesn't implement CloseWrite method")
	}
	_, err := io.WriteString(c, message)
	if err != nil {
		return fmt.Errorf("error writing to conn: %w", err)
	}

	return nil
}
