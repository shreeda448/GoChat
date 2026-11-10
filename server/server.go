package main

import (
	"fmt"
	"io"
	"log"
	"net"
)

const (
	addr         = "127.0.0.1:8080"
	mxBufSiz int = 4096
)

func handleOneClient(cli net.Conn) {
	defer cli.Close()
	buf := make([]byte, mxBufSiz)
	for {
		n, err := cli.Read(buf)
		if n > 0 {
			if _, werr := cli.Write(buf[:n]); werr != nil {
				log.Println("(error) write:", werr)
				return
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Println("(error) read:", err)
			}
			return
		}
	}
}

func main() {
	fmt.Println("GoChat server starting.....")
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
		go handleOneClient(cli)
	}
}
