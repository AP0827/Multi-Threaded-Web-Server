package core

import (
	"bufio"
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
	_, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Read error:", err)
		return
	}

	respond(conn)
}
func respond(conn net.Conn) {
	response := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Length: 5\r\n" +
		"\r\n" +
		"Hello"

	conn.Write([]byte(response))
}
