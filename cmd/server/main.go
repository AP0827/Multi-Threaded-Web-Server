package main

import (
	"MTWS/config"
	"MTWS/pool"
	"MTWS/security/ratelimiter"
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

	/* make buffered job pool queue here. */
	jobs := make(chan pool.Job, config.JobQueueSize)
	pool.StartWorkerPool(config.WorkerPoolSize, jobs)

	log.Println("Server running on port:8080!")

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Accept error : ", err)
			continue
		}

		remoteAddr := conn.RemoteAddr().String()
		clientIP, _, err := net.SplitHostPort(remoteAddr)
		if err != nil {
			clientIP = remoteAddr
		}

		if limiter.Allow(clientIP) {
			jobs <- pool.Job{Conn: conn}
			continue
		}

		log.Printf("Status 429: request blocked by token bucket policy for %s", clientIP)
		response := "HTTP/1.1 429 Too Many Requests\r\n" +
			"Content-Type: text/plain\r\n" +
			"Content-Length: 17\r\n" +
			"\r\n" +
			"Too Many Requests"
		if _, err := conn.Write([]byte(response)); err != nil {
			log.Println("Write error:", err)
		}
		conn.Close()
	}
}
