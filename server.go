package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

type Server struct {
	// internal
	addr    string
	logfile string

	// clients info, sync map has all locking
	clients *sync.Map

	// various channels
	msgchan chan *Message
	addchan chan *Client
	rmchan  chan *Client

	// REST API Server
	restserver *RestServer
}

func NewServer(addr, http_addr, logfile string) *Server {
	s := &Server{
		addr:    addr,
		logfile: logfile,
		clients: new(sync.Map),
		msgchan: make(chan *Message),
		addchan: make(chan *Client),
		rmchan:  make(chan *Client),
	}

	// Create rest api server
	restserver := NewRestServer(http_addr, logfile, s.msgchan)
	s.restserver = restserver
	return s
}

func (s *Server) Run(ctx context.Context) (err error) {
	// Collect errors
	var wg sync.WaitGroup
	defer wg.Done()
	errors := make(chan error, 2)

	// Server
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting TCP server on '%v'", s.addr)
		errors <- s.StartServer(ctx)
	}()

	// REST Server
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting HTTP server on '%v'", s.restserver.http_addr)
		errors <- s.restserver.StartHttpServer(ctx)
	}()

	select {
	case err = <-errors:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) StartServer(ctx context.Context) error {
	// Setup server logging
	f, err := os.OpenFile(s.logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	// Output everything in the log file
	log.SetOutput(f)

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	// Log time when server started listening
	log.Printf("Server has started on '%v'", s.addr)

	// Go handle events
	go s.handleEvents()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}

		// Handle connections and printout errors
		go func() {
			err := s.handleConnection(conn)
			if err != nil {
				log.Print(err)
			}
		}()
	}

	return nil
}

// Do basic auth for nickname.
func (s *Server) sendAuth(c net.Conn) (nick string, err error) {
	// Write back to the client
	_, err = io.WriteString(c, "Provide your nickname: ")
	if err != nil {
		return "", err
	}

	// Read nickname
	reader := bufio.NewReader(c)
	nickbytes, _, err := reader.ReadLine()
	if err != nil {
		return "", err
	}

	return string(nickbytes), nil
}

// Server handles new incoming connections.
func (s *Server) handleConnection(c net.Conn) error {
	defer c.Close()

	// Do auth for client
	nickname, err := s.sendAuth(c)
	if err != nil {
		return err
	}

	// Check for empty nickname
	if strings.TrimSpace(nickname) == "" {
		// Try best writing back to the client
		io.WriteString(c, "Empty nickname, BYE\n")
		msg := fmt.Sprintf("Connected client '%v' provided empty nickname, ignore",
			c.RemoteAddr())
		return errors.New(msg)
	}

	// Make new client
	client := NewClient(c, nickname, make(chan *Message))

	// Register one
	s.addchan <- client

	// Welcome new user
	io.WriteString(c, fmt.Sprintf("Welcome, %s!\n", client.nickname))

	// Do input/output
	go func() {
		err := client.Receive(s.msgchan)
		// If receive failed, client disconnected, remove client info
		if err != nil {
			s.rmchan <- client
		}
	}()
	return client.Send(client.ch)
}

func (s *Server) handleEvents() {
	// Loop for various events
	for {
		select {
		case msg := <-s.msgchan:
			log.Printf("New message: '%+v'\n", strings.TrimSpace(msg.Content))
			// sync.map Range is a bit hairy
			s.clients.Range(func(key interface{}, value interface{}) bool {
				v := value.(*Client)
				// Broadcast message to everyone, except the origin
				if v.nickname != msg.From {
					go func(msgch chan<- *Message) {
						msgch <- msg
					}(v.ch)
				}
				return true
			})
		case client := <-s.addchan:
			log.Printf("New client connected: '%v' with '%v'\n",
				client.nickname, client.conn.RemoteAddr())
			s.clients.Store(client.nickname, client)
		case client := <-s.rmchan:
			log.Printf("Client disconnects: '%v' with '%v'\n",
				client.nickname, client.conn.RemoteAddr())
			s.clients.Delete(client.nickname)
		}
	}
}
