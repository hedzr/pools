// Copyright © 2019 Hedzr Yeh.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/hedzr/pools/connpool"
	"github.com/hedzr/pools/examples/tool"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// To test this example:
// $ ulimit -n 20000
// $ go run examples/connpooldemo/main.go
// $ go run examples/connpooldemo/main.go -ping-time 2s -pool-size 500

const addr = ":1180"

var (
	// port           = flag.Int("port", 50001, "listening port")
	poolSize         = flag.Int("pool-size", 10, "size of clients connect pool")
	totalRequests    = flag.Int("total-clients", 3000, "how many clients to be simulated")
	keepAliveTimeout = flag.Duration("ping-time", 3*time.Second, "ping period (default is 3s)")
)

func main() {
	flag.Parse()

	appExit := make(chan struct{})
	startAt := time.Now()

	// appExit := make(chan os.Signal)
	// signal.Notify(appExit, os.Interrupt, os.Kill, syscall.SIGUSR1, syscall.SIGUSR2)

	closer := startServer(appExit, addr)
	defer closer.Close()

	pool := connpool.New(*poolSize,
		connpool.WithWorkerDialer(newWorkerWithOpts(WithClientKeepAliveTimeout(*keepAliveTimeout))),
		connpool.WithKeepAliveInterval(*keepAliveTimeout),
		connpool.WithBlockIfCantBorrow(true),
	)
	defer pool.Close()

	go startClients(appExit, pool, *totalRequests)

	// <-appExit

	<-appExit
	fmt.Println(time.Now().Sub(startAt).Seconds(), " seconds")
}

// ----------------------------------------------------

func startServer(appExit chan struct{}, addr string) (closer io.Closer) {
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
	exitingFlag bool
	done        chan struct{}
	l           net.Listener
	writeCh     chan []byte
}

func (s *serverImpl) Close() (err error) {
	s.exitingFlag = true
	if s.l != nil {
		s.l.Close()
		s.l = nil
	}
	if s.done != nil {
		close(s.done)
		s.done = nil
	}
	if s.writeCh != nil {
		close(s.writeCh)
		s.writeCh = nil
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

// ----------------------------------------------------

func startClients(appExit chan struct{}, pool connpool.Pool, howMany int) {
	var wg sync.WaitGroup
	for i := 0; i < howMany; i++ {
		wg.Add(1)
		go oneClient(&wg, pool, i)
	}
	wg.Wait()
	appExit <- struct{}{}
}

func oneClient(wg *sync.WaitGroup, pool connpool.Pool, idx int) {
	if c, ok := pool.Borrow().(*clientSample); ok {
		c.index = idx
		// fmt.Printf("    |%6d| pool actived: %v\n", idx, pool.Borrowed())
		if c != nil {
			if _, err := c.LongOper(pool); err != nil {
				log.Fatalf("error in Oper(): %v", err)
			} else {
				pool.Return(c)
			}
		} else {
			log.Printf("[WARN] |%6d| can't borrow", idx)
		}
		wg.Done()
	}
}

func newWorker() (w connpool.Worker, err error) {
	w, err = newWorkerWithOpts()()
	return
}

func newWorkerWithOpts(opts ...ClientSampleOpt) connpool.Dialer {
	return func() (w connpool.Worker, err error) {
		c := &clientSample{
			keepAliveTimeout: 15 * time.Second,
			sendCh:           make(chan string),
			doneCh:           make(chan struct{}),
		}
		err = c.open()
		if err == nil {
			for _, opt := range opts {
				opt(c)
			}
		}
		w = c
		return
	}
}

func WithClientKeepAliveTimeout(d time.Duration) ClientSampleOpt {
	return func(sample *clientSample) {
		sample.keepAliveTimeout = d
	}
}

func WithClientIndex(i int) ClientSampleOpt {
	return func(sample *clientSample) {
		sample.index = i
	}
}

type clientSample struct {
	keepAliveTimeout time.Duration
	conn             net.Conn
	index            int
	sendCh           chan string
	doneCh           chan struct{}
}

type ClientSampleOpt func(*clientSample)

func (c *clientSample) Close() (err error) {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	return
}

func (c *clientSample) open() (err error) {
	c.conn, err = net.Dial("tcp", addr)
	go c.worker()
	go c.readWorker()
	return
}

func (c *clientSample) worker() {
	var n int
	var err error
	for {
		select {
		case <-c.doneCh:
			return
		case s := <-c.sendCh:
			n, err = c.conn.Write([]byte(s))
			if n < 0 || err != nil {
				if err == io.EOF {
					return
				}
				log.Printf("err: %v", err)
				return
			}
		}
	}
}

func (c *clientSample) readWorker() {
	buf := make([]byte, 4096)
	for {
		n, err := c.conn.Read(buf)
		if n < 0 || err != nil {
			if err == io.EOF {
				return
			} else if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			log.Printf("err: %v", err)
			return
		}
		b := buf[:n]
		if string(b) == "PONG" {
			log.Printf("PONG for # %v", c.index)
		}
	}
}

func (c *clientSample) Tick(tick time.Time) (err error) {
	c.sendCh <- "PING"
	return
}

func (c *clientSample) LongOper(pool connpool.Pool) (n int, err error) {
	// buf := make([]byte, 128)
	// n, err = c.conn.Write(buf)
	// n, err = c.conn.Read(buf[:n])
	c.sendCh <- tool.RandomStringPure(128)
	time.Sleep(time.Duration(100+rand.Intn(16000)) * time.Millisecond)
	log.Printf("    |%6d| longOper() end | pool actived: %v |", c.index, pool.Borrowed())
	return
}
