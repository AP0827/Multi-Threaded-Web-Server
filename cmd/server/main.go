package main

import (
	"MTWS/config"
	"MTWS/pool"
	"MTWS/security/ratelimiter"
	"fmt"
	"log"
	"net"
)

var limiter = ratelimiter.New(config.RateLimitRate, config.RateLimitCapacity)

func main() {
	l, err := net.Listen("tcp", config.ServerAddress)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	jobs := make(chan pool.Job, config.JobQueueSize)
	pool.StartWorkerPool(config.WorkerPoolSize, jobs)

	log.Printf("Server listening on %s", config.ServerAddress)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}

		handleConnection(conn, jobs)
	}
}

func handleConnection(conn net.Conn, jobs chan pool.Job) {
	clientIP := remoteIP(conn)

	if !limiter.Allow(clientIP) {
		log.Printf("Status %d: request blocked by token bucket policy for %s", 429, clientIP)
		if err := writeResponseAndClose(conn, 429, "Too Many Requests"); err != nil {
			log.Println("write error:", err)
		}
		return
	}

	select {
	case jobs <- pool.Job{Conn: conn}:
		return
	default:
		log.Printf("Status %d: queue full, request rejected for %s", 503, clientIP)
		if err := writeResponseAndClose(conn, 503, "Service Unavailable"); err != nil {
			log.Println("write error:", err)
		}
	}
}

func remoteIP(conn net.Conn) string {
	remoteAddr := conn.RemoteAddr().String()
	clientIP, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return clientIP
}

func writeResponseAndClose(conn net.Conn, statusCode int, body string) error {
	response := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, httpStatusText(statusCode)) +
		"Content-Type: text/plain\r\n" +
		fmt.Sprintf("Content-Length: %d\r\n", len(body)) +
		"\r\n" +
		body

	_, err := conn.Write([]byte(response))
	conn.Close()
	return err
}

func httpStatusText(statusCode int) string {
	switch statusCode {
	case 429:
		return "Too Many Requests"
	case 503:
		return "Service Unavailable"
	default:
		return "OK"
	}
}
