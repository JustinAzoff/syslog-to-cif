package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	cif "github.com/JustinAzoff/cifsdk-go"
)

//For pushing up to main using channels?
type Notice struct {
	Note string
	Msg  string
	Src  string
	Dst  string
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
		var notice Notice
		buf := scanner.Bytes()
		braceOffset := bytes.IndexByte(buf, '{')
		if braceOffset == -1 {
			log.Printf("No open brace found in msg: %q", scanner.Text())
			continue
		}
		jsonBytes := buf[braceOffset:]
		err := json.Unmarshal(jsonBytes, &notice)
		if err != nil {
			log.Printf("Error parsing json from connection: %v", err)
			continue
		}
		noticeChan <- notice
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from connection: %v", err)
	}
}

func createIndicators(c *cif.Client, notices []Notice) error {
	log.Printf("Creating %d indicators", len(notices))
	var indicators cif.IndicatorList
	for _, n := range notices {
		i := cif.Indicator{
			Indicator:   n.Src,
			Description: fmt.Sprintf("%s: %s", n.Note, n.Msg),
			Tags:        []string{"bro"},
		}
		log.Printf("Creating %s %s", i.Indicator, i.Description)
		indicators = append(indicators, i)
	}
	err := c.CreateIndicators(indicators)
	if err != nil {
		log.Printf("Error creating indicators: %v", err)
	}
	return nil
}

func receive(addr string, port int, cifEndpoint string) {
	noticeBuffer := make([]Notice, 0, 100) // len()=0, cap()=100
	noticeChan := make(chan Notice, 100)
	go listen(addr, port, noticeChan)

	c := &cif.Client{
		Endpoint: cifEndpoint,
		Token:    os.Getenv("CIF_TOKEN"),
		Debug:    true,
	}

	ticker := time.NewTicker(5 * time.Second)
	var send bool
	for {
		send = false
		select {
		case <-ticker.C:
			if len(noticeBuffer) > 0 {
				send = true
			}
		case note := <-noticeChan:
			noticeBuffer = append(noticeBuffer, note)
			if len(noticeBuffer) > 50 {
				send = true
			}
		}
		if send {
			createIndicators(c, noticeBuffer)
			noticeBuffer = noticeBuffer[:0]
		}
	}
}

func main() {
	var port int
	var addr string
	var cif string
	flag.StringVar(&cif, "endpoint", "http://127.0.0.1:5000", "CIF Endpoint")
	flag.StringVar(&addr, "addr", "0.0.0.0", "Address to listen on")
	flag.IntVar(&port, "port", 9000, "Port to listen on")
	flag.Parse()
	receive(addr, port, cif)
}
