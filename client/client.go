package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func readServ(cli net.Conn, done chan struct{}) {
	defer close(done)
	_, err := io.Copy(os.Stdout, cli)
	if err != nil && !errors.Is(err, net.ErrClosed) {
		log.Printf("(error) failed to read from the server: %v\n", err)
	}
}

func readInput(conn net.Conn) {
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := sc.Text()
		if line == "exit" || line == "quit" {
			fmt.Println("Exiting GoChat.....")
			break
		}
		if _, err := fmt.Fprintln(conn, line); err != nil {
			log.Println("(error) send:", err)
			break
		}
	}
	conn.Close()
}

func main() {
	fmt.Println("GoChat client starting.....")
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	fmt.Println("Connected to server! Type your message and press Enter:")
	done := make(chan struct{})
	go readServ(conn, done)
	go readInput(conn)
	<-done
}
