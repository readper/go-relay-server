package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

const (
	connType = "tcp"
	timeout  = 10 * time.Second

	quitCmd = "quit"
)

var (
	port = flag.Int("port", 23, "port to listen and serve telnet server")
	host = flag.String("host", "", "host to listen")
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

		msg = strings.Replace(msg, "\r\n", "\n", -1)
		msg = strings.Replace(msg, "\r", "\n", -1)

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

}
