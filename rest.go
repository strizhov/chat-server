package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type RestServer struct {
	http_addr string
	logfile   string

	// Channel for new messages
	msgchan chan *Message
}

func NewRestServer(http_addr, logfile string, msgchan chan *Message) *RestServer {
	return &RestServer{
		http_addr: http_addr,
		logfile:   logfile,
		msgchan:   msgchan,
	}
}

// REST Handler
func (rst *RestServer) StartHttpServer(ctx context.Context) error {
	http.HandleFunc("/", rst.handler)
	return http.ListenAndServe(rst.http_addr, nil)
}

func (rst *RestServer) handler(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method == http.MethodGet {
		rst.handleGet(w, r)
	} else if r.Method == http.MethodPost {
		rst.handlePost(w, r)
	} else {
		io.WriteString(w, "Sorry "+r.Method+" is not supported.")
	}
}

// HTTP POST Requests
func (rst *RestServer) handlePost(w http.ResponseWriter, r *http.Request) {
	// Parse json from http body
	msg := &Message{}
	err := json.NewDecoder(r.Body).Decode(msg)
	if err != nil {
		log.Printf("Unable to parse json object: %+v\n", err)
		return
	}

	// Check if values present
	if msg.From == "" || msg.Content == "" {
		log.Println("Received empty json object, ignore.")
		return
	}

	// Append time and nickname
	fmt_content := fmt.Sprintf("%s %s: %s\n",
		time.Now().Format(time.ANSIC), msg.From, msg.Content)
	msg.Content = fmt_content

	// Broadcast to everyone
	rst.msgchan <- msg

	// Respond with OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// HTTP GET Requests
func (rst *RestServer) handleGet(w http.ResponseWriter, r *http.Request) {
	// Open file for read
	file, err := os.Open(rst.logfile)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()

	// Start reading from the file with a reader.
	reader := bufio.NewReader(file)

	// Write OK to requester
	w.WriteHeader(http.StatusOK)

	// Do best sending each matching line
	var line string
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			break
		}
		// The log format is:
		// New message: 'Sun Dec  2 17:25:04 2018 bob: hello'
		// Send only these messages, ignore the rest
		if strings.Contains(line, "New message: ") {
			// Look for single quotes and return result back
			first_idx := strings.Index(line, "'")
			last_idx := strings.LastIndex(line, "'")
			ret := line[first_idx+1:last_idx] + "\n"
			w.Write([]byte(ret))
		}
	}
	return
}
