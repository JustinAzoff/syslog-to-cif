package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"time"
)

//For pushing up to main using channels?
type Notice struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

func listen(addr string, port int, noticeChan chan Notice) {
	bind := fmt.Sprintf("%s:%d", addr, port)
	log.Printf("Listening on %s", bind)
	l, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatalf("Error listening: %v", err)
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalf("Error accepting: %v", err)
		}
		log.Printf("New connection from %s", conn.RemoteAddr())
		go handleLog(conn, noticeChan)
	}
}

func handleLog(conn net.Conn, noticeChan chan Notice) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		fmt.Println("Got " + scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from connection: %v", err)
	}
}

func receive(addr string, port int) {
	noticeBuffer := make([]Notice, 0, 100) // len()=0, cap()=100
	noticeChan := make(chan Notice, 100)
	go listen(addr, port, noticeChan)

	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			for _, n := range noticeBuffer {
				log.Print(n)
			}
			noticeBuffer = noticeBuffer[:0]
		case note := <-noticeChan:
			noticeBuffer = append(noticeBuffer, note)
		}
	}
}

func main() {
	var port int
	var addr string
	flag.StringVar(&addr, "addr", "0.0.0.0", "Address to listen on")
	flag.IntVar(&port, "port", 9000, "Port to listen on")
	flag.Parse()
	receive(addr, port)
}
