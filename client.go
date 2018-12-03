package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"time"
)

type Client struct {
	conn     net.Conn
	nickname string
	ch       chan *Message
}

func NewClient(conn net.Conn, nickname string, ch chan *Message) *Client {
	client := &Client{
		conn:     conn,
		nickname: nickname,
		ch:       ch,
	}
	return client
}

// Receive message from Message channel and show it
func (c *Client) Receive(ch chan<- *Message) error {
	reader := bufio.NewReader(c.conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		// Print out local time with nickname+message
		fmt_content := fmt.Sprintf("%s %s: %s",
			time.Now().Format(time.ANSIC), c.nickname, line)
		message := &Message{
			From:    c.nickname,
			Content: fmt_content,
		}

		ch <- message
	}
}

// Send message to Message channel
func (c *Client) Send(ch <-chan *Message) error {
	for msg := range ch {
		_, err := io.WriteString(c.conn, msg.Content)
		if err != nil {
			return err
		}
	}
	return nil
}
