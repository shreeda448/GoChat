package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

const (
	addr = "127.0.0.1:8080"
)

type Hub struct {
	mu      sync.Mutex
	clients map[string]*Client
}

type Client struct {
	username string
	conn     net.Conn
	out      chan string
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]*Client),
	}
}

func (c *Client) deliver(msg string) {
	select {
	case c.out <- msg:
	default:
	}
}

func (h *Hub) Register(cli net.Conn, username string) (*Client, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, taken := h.clients[username]; taken {
		return nil, errors.New("username already taken")
	}
	c := NewClient(cli, username)
	h.clients[username] = c
	return c, nil
}

func (h *Hub) UnRegister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.username] == c {
		delete(h.clients, c.username)
		close(c.out)
	}
}

func (h *Hub) Broadcast(from string, msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for name, c := range h.clients {
		if name != from {
			c.deliver(msg)
		}
	}
}

func (h *Hub) Private(from string, to string, msg string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	dest, ok := h.clients[to]
	if !ok {
		return fmt.Errorf("user %v not found", to)
	}
	dest.deliver(fmt.Sprintf("(private) %s: %s", from, msg))
	return nil
}

func NewClient(cli net.Conn, username string) *Client {
	o := make(chan string, 16)
	newClient := Client{
		username: username,
		conn:     cli,
		out:      o,
	}
	return &newClient
}

func (c *Client) Drain() {
	for msg := range c.out {
		if _, err := fmt.Fprintln(c.conn, msg); err != nil {
			log.Println("(error) failed to drain", err)
			c.conn.Close()
			return
		}
	}
}

func handleOneClient(hub *Hub, cli net.Conn) {
	defer cli.Close()
	sc := bufio.NewScanner(cli)
	var c *Client
	for c == nil {
		fmt.Fprint(cli, "Enter username: ")
		if !sc.Scan() {
			return
		}
		username := strings.TrimSpace(sc.Text())
		if username == "" || strings.ContainsAny(username, " \t") {
			fmt.Fprintln(cli, "Username must be non-empty and contain no spaces.")
			continue
		}
		var err error
		if c, err = hub.Register(cli, username); err != nil {
			fmt.Fprintln(cli, err)
		}
	}
	hub.Broadcast(c.username, fmt.Sprintf("%s has joined the chat", c.username))
	go c.Drain()
	defer hub.Broadcast(c.username, fmt.Sprintf("%s left the chat", c.username))
	defer hub.UnRegister(c)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/msg ") {
			parts := strings.SplitN(line, " ", 3)
			if len(parts) < 3 {
				c.deliver("usage: /msg username message")
				continue
			}
			if err := hub.Private(c.username, parts[1], parts[2]); err != nil {
				c.deliver(err.Error())
			}
			continue
		} else {
			hub.Broadcast(c.username, fmt.Sprintf("[%s] %s", c.username, line))
		}
	}
}

func main() {
	fmt.Println("GoChat server starting.....")
	hub := NewHub()
	lsnr, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer lsnr.Close()
	for {
		cli, err := lsnr.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleOneClient(hub, cli)
	}
}
