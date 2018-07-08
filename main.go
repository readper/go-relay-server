package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	connType      = "tcp"
	timeout       = 10 * time.Second
	delayDuration = 100 * time.Microsecond
	quitCmd       = "quit"
)

var (
	port           = flag.Int("port", 23, "port to listen and serve telnet server")
	host           = flag.String("host", "", "host to listen")
	limitPerSecond = flag.Int("limit-per-second", -1, "api request will be delayed when limit exceeded, disabled when <= 0")

	mutex        = &sync.Mutex{}
	requestCount = 0
)

func main() {
	addr, err := net.ResolveTCPAddr(connType, fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		log.Panicf("host %s and port %d parse failed, error: %s", *host, *port, err.Error())
	}
	listener, err := net.ListenTCP(connType, addr)
	if err != nil {
		log.Panicf("Listen to %s:%d failed, error: %s", *host, *port, err.Error())
	}
	defer listener.Close()
	log.Printf("start listening %s", addr.String())

	// reset count every second
	go func() {
		tick := time.Tick(time.Second)
		// loop:
		for {
			select {
			case <-tick:
				mutex.Lock()
				requestCount = 0
				mutex.Unlock()
			}
		}
	}()

	for {
		tcpConn, err := listener.AcceptTCP()
		if err != nil {
			log.Panicf("acceptTCP failed, error: %s", err.Error())
		}
		err = tcpConn.SetDeadline(time.Now().Add(timeout))
		if err != nil {
			log.Panicf("set connection deadline failed, error: %s", err.Error())
		}
		go handleConnection(tcpConn)
	}
}

func handleConnection(conn *net.TCPConn) {
	remoteAddr := conn.RemoteAddr()
	defer log.Printf("connection close from %s", remoteAddr.String())
	log.Printf("new connect from %s", remoteAddr.String())
	for {
		// read data from client
		buffer := make([]byte, 1024)
		length, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				log.Println("Read encounter EOF, normal end")
			} else {
				log.Printf("read fail, error: %s", err.Error())
			}
			conn.Close()
			return
		}
		msg := string(buffer)
		log.Printf("Got Message: %s size: %d from %s ", msg, length, remoteAddr.String())

		// reset timeout
		err = conn.SetDeadline(time.Now().Add(timeout))
		if err != nil {
			log.Panicf("reset connection deadline failed, error: %s", err.Error())
		}

		// handle different end of line
		msg = strings.Replace(msg, "\r\n", "\n", -1)
		msg = strings.Replace(msg, "\r", "\n", -1)

		// handle each line
		inputs := strings.Split(msg, "\n")
		for _, input := range inputs {
			if input == "" {
				continue
			}
			if input == quitCmd {
				conn.Close()
				return
			}
			handleInput(input)
		}
	}
}

// each line will be a input
func handleInput(input string) {
	// check limit
	for {
		if requestCount >= *limitPerSecond {
			time.Sleep(delayDuration)
		}
	}
	mutex.Lock()
	requestCount++
	mutex.Unlock()

	// do request
}
