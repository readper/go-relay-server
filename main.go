package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type stat int

const (
	connType      = "tcp"
	timeout       = 10 * time.Second
	delayDuration = 100 * time.Microsecond
	quitCmd       = "quit"
	requestURL    = "https://readper.asuscomm.com:312/test"

	statsCurrentRequest stat = iota + 1
	statsTotalRequest
	statsCurrentConnection
	statsTotalConnection
)

var (
	port           = flag.Int("port", 23, "port to listen and serve telnet server")
	host           = flag.String("host", "", "host to listen")
	limitPerSecond = flag.Int("limit-per-second", -1, "api request will be delayed when limit exceeded, disabled when <= 0")

	mutex        = &sync.Mutex{}
	requestCount = 0

	stats = map[stat]int{statsCurrentConnection: 0, statsCurrentRequest: 0, statsTotalConnection: 0, statsTotalRequest: 0}
)

func main() {
	flag.Parse()
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

	// Stats report
	go func() {
		tickStat := time.Tick(5 * time.Second)
		for {
			select {
			case <-tickStat:
				log.Printf("Current Connection: %d", stats[statsCurrentConnection])
				log.Printf("Total Connection: %d", stats[statsTotalConnection])
				log.Printf("Current Request: %d", stats[statsCurrentRequest])
				log.Printf("Total Request: %d", stats[statsTotalRequest])
			}
		}
	}()

	// ratelimiter work only if flag > 0
	if *limitPerSecond > 0 {
		// reset count every second
		go func() {
			tick := time.Tick(time.Second)
			for {
				select {
				case <-tick:
					mutex.Lock()
					stats[statsCurrentRequest] = 0
					mutex.Unlock()
				}
			}
		}()
	}

	for {
		tcpConn, err := listener.AcceptTCP()
		ctx := context.WithValue(context.Background(), "requestID", randStr(32))
		if err != nil {
			log.Panicf("acceptTCP failed, error: %s", err.Error())
		}
		err = tcpConn.SetDeadline(time.Now().Add(timeout))
		if err != nil {
			log.Panicf("set connection deadline failed, error: %s", err.Error())
		}
		go handleConnection(ctx, tcpConn)
	}
}

func handleConnection(ctx context.Context, conn *net.TCPConn) {
	remoteAddr := conn.RemoteAddr()
	mutex.Lock()
	stats[statsCurrentConnection]++
	stats[statsTotalConnection]++
	mutex.Unlock()
	defer func() {
		mutex.Lock()
		stats[statsCurrentConnection]--
		mutex.Unlock()
		log.Printf("connection close from %s, requestID: %s", remoteAddr.String(), ctx.Value("requestID"))
	}()

	log.Printf("new connect from %s, requestID: %s", remoteAddr.String(), ctx.Value("requestID"))
	for {
		// read data from client
		buffer := make([]byte, 1024)
		length, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				log.Println("Read encounter EOF, normal end, requestID: %s", ctx.Value("requestID"))
			} else {
				log.Printf("read fail, error: %s, requestID: %s", err.Error(), ctx.Value("requestID"))
			}
			conn.Close()
			return
		}
		msg := string(buffer)
		msg = msg[:length]
		log.Printf("Got Message: %s size: %d from %s, requestID: %s", msg, length, remoteAddr.String(), ctx.Value("requestID"))

		// reset timeout
		err = conn.SetDeadline(time.Now().Add(timeout))
		if err != nil {
			log.Panicf("reset connection deadline failed, error: %s, requestID: %s", err.Error(), ctx.Value("requestID"))
		}

		// handle different end of line
		msg = strings.Replace(msg, "\r\n", "\n", -1)
		msg = strings.Replace(msg, "\r", "\n", -1)

		// handle each line
		inputs := strings.Split(msg, "\n")

		for _, input := range inputs {
			if input == quitCmd {
				conn.Close()
				return
			}
			returnBody, err := handleInput(ctx, input)
			if err != nil {
				log.Printf("api request fail, input: %s, error: %s, requestID: %s", input, err.Error(), ctx.Value("requestID"))
				continue
			}
			conn.Write([]byte(fmt.Sprintf("\ninput: %s\nreturn: \n", input)))
			conn.Write(returnBody)
			conn.Write([]byte("\n"))
		}
	}
}

// each line will be a input
func handleInput(ctx context.Context, input string) ([]byte, error) {
	if *limitPerSecond > 0 {
		// check limit
		for {
			mutex.Lock()
			currentRequest := stats[statsCurrentRequest]
			mutex.Unlock()
			if currentRequest >= *limitPerSecond {
				time.Sleep(delayDuration)
			} else {
				break
			}
		}
		mutex.Lock()
		stats[statsCurrentRequest]++
		stats[statsTotalRequest]++
		mutex.Unlock()
	}
	// do request
	httpResp, err := http.Post(requestURL, "application/x-www-form-urlencoded", strings.NewReader(fmt.Sprintf("input=%s", input)))
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func randStr(n int) string {
	charSet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefijklmnopqrstuvwxyz1234567890"
	output := ""
	for i := 0; i < n; i++ {
		output += string(charSet[rand.Int63n(int64(len(charSet)))])
	}
	return output
}
