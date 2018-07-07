package main

import (
	"flag"
	"fmt"
	"log"
	"net"
)

const (
	connType = "tcp"
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

	for {
		tcpConn, err := listener.AcceptTCP()
		if err != nil {
			log.Panicf("acceptTCP failed, error: %s", err.Error())
		}
		go handleConnection(tcpConn)
	}
}

func handleConnection(conn *net.TCPConn) {
	remoteAddr := conn.RemoteAddr()
	defer log.Printf("connection close from %s", remoteAddr.String())
	log.Printf("new connect from %s", remoteAddr.String())
	buffer := make([]byte, 1024)
	length, err := conn.Read(buffer)
	if err != nil {
		log.Printf("read fail, error: %s", err.Error())
		conn.Close()
	}

	log.Printf("Got Message: %s size: %d from %s ", string(buffer), length, remoteAddr.String())

}
