package core

import (
	mtwshttp "MTWS/http"
	"bufio"
	"fmt"
	"log"
	"net"
	"time"
)

func HandleConnection(conn net.Conn) {
	defer conn.Close()

	log.Println("New request from", conn.RemoteAddr())

	// Prevent slowloris-style hanging
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	reader := bufio.NewReader(conn)

	// Read raw request (just first line for now)
	// _, err := reader.ReadString('\n')
	// if err != nil {
	// 	log.Println("Read error:", err)
	// 	return
	// }

	req, err := mtwshttp.ParseRequest(reader)
	if err != nil {
		log.Println("Read error:", err)
		respondBadRequest(conn)
		return
	}

	respond(conn, req)
}

func respond(conn net.Conn, req *mtwshttp.Request) {
	log.Printf("Request method=%s", req.Method())
	log.Printf("Request path=%s", req.Path())
	log.Printf("Request version=%s", req.Version())

	body := fmt.Sprintf("Hello path=%s\n", req.Path())

	if path == "/" {
		body = "Hello world\n"

	}

	response := req.Version() + " 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		fmt.Sprintf("Content-Length: %d\r\n", len(body)) +
		"\r\n" +
		body

	if _, err := conn.Write([]byte(response)); err != nil {
		log.Println("Write error:", err)
	}
}

func respondBadRequest(conn net.Conn) {
	response := "HTTP/1.1 400 Bad Request\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Length: 11\r\n" +
		"\r\n" +
		"Bad Request"

	if _, err := conn.Write([]byte(response)); err != nil {
		log.Println("Write error:", err)
	}
}
