// Copyright © 2019 Hedzr Yeh.

package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"
)

// ----------------------------------------------------

// New new a tcp server
func New(appExit chan struct{}, addr string) (closer io.Closer) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Error listening: ", err)
		os.Exit(1)
	}

	fmt.Println("Listening on ", addr)

	server := &serverImpl{
		done:    make(chan struct{}),
		l:       l,
		writeCh: make(chan []byte, 128),
	}
	go server.run()
	closer = server
	return
}

type serverImpl struct {
	exited      int32
	exitingFlag bool
	done        chan struct{}
	l           net.Listener
	writeCh     chan []byte
}

func (s *serverImpl) Close() (err error) {
	if atomic.LoadInt32(&s.exited) == 0 {
		atomic.AddInt32(&s.exited, 1)
		s.exitingFlag = true

		close(s.done)

		s.l.Close()

		close(s.writeCh)
	}
	return
}

func (s *serverImpl) run() {
	for {
		conn, err := s.l.Accept()
		if err != nil {
			if s.exitingFlag {
				return
			}
			fmt.Println("Error accepting: ", err)
			continue
		}
		fmt.Printf("Accepted connection %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())

		// todo use fixed-pool to limit the high-bound of goroutines
		go s.handleRequest(conn, time.Now().UTC(), s.done)
	}
}

func (s *serverImpl) handleRequest(conn net.Conn, tick time.Time, done chan struct{}) {
	reader := bufio.NewReader(conn)
	writer := conn // bufio.NewWriter(conn)
	go s.hubWriting(writer)

	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				// log.Printf("conn(from: %v) read i/o eof found. closing ", conn.RemoteAddr())
			} else {
				log.Println("Error reading: ", err)
			}
			return
		}

		data := buf[:n]
		if string(data) == "PING" {
			s.writeCh <- []byte("PONG")
			continue
		}

		s.writeCh <- data
	}
}

func (s *serverImpl) hubWriting(writer io.Writer) {
	for {
		select {
		case <-s.done:
			return
		case data := <-s.writeCh:
			nn, err := writer.Write(data)
			if err != nil {
				if err == io.EOF {
					log.Printf("write i/o eof found. closing ")
				} else {
					log.Printf("Error writing %v bytes: %v", nn, err)
				}
				return
			}
			// log.Printf("and %v bytes reply", nn)
		}
	}
}
