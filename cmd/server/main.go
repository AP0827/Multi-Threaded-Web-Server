package main

import (
	"MTWS/config"
	"MTWS/core"
	mtwshttp "MTWS/http"
	"MTWS/pool"
	"MTWS/security/ratelimiter"
	"fmt"
	"log"
	"net"
	"strings"
)

func main() {
	l, err := net.Listen("tcp", config.ServerAddress)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	/* make buffered job pool queue here. */
	jobs := make(chan pool.Job, config.JobQueueSize)
	pool.StartWorkerPool(config.WorkerPoolSize, jobs)
	router := buildRouter()
	limiter := ratelimiter.New(config.RateLimitRate(), config.RateLimitCapacity())
	rateLimitEnabled := config.RateLimitEnabled()

	log.Println("Server running on port:8080!")
	if rateLimitEnabled {
		log.Printf("Rate limiter enabled: rate=%.2f capacity=%.2f", config.RateLimitRate(), config.RateLimitCapacity())
	} else {
		log.Println("Rate limiter disabled by environment")
	}

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

		if !rateLimitEnabled || limiter.Allow(clientIP) {
			jobs <- pool.Job{Conn: conn, Router: router}
			continue
		}

		log.Printf("Status 429: request blocked by token bucket policy for %s", clientIP)
		if err := core.NewResponseWriter(conn, "HTTP/1.1").WriteText(core.StatusTooManyRequests, ""); err != nil {
			log.Println("Write error:", err)
		}
		conn.Close()
	}
}

func buildRouter() *core.Router {
	router := core.NewRouter()

	router.Handle("/", func(w *core.ResponseWriter, req *mtwshttp.Request) {
		body := "MTWS baseline server is running\n"
		if err := w.WriteText(core.StatusOK, body); err != nil {
			log.Println("Write error:", err)
		}
	})

	router.Handle("/health", func(w *core.ResponseWriter, req *mtwshttp.Request) {
		body := "ok\n"
		if err := w.WriteText(core.StatusOK, body); err != nil {
			log.Println("Write error:", err)
		}
	})

	router.Handle("/submit", func(w *core.ResponseWriter, req *mtwshttp.Request) {
		body := "MTWS baseline server is running\n"
		if err := w.WriteText(core.StatusOK, body); err != nil {
			log.Println("Write error:", err)
		}
	})

	router.Handle("/search", func(w *core.ResponseWriter, req *mtwshttp.Request) {
		_, query, _ := strings.Cut(req.Path(), "?")
		body := fmt.Sprintf("query=%s\n", query)
		if err := w.WriteText(core.StatusOK, body); err != nil {
			log.Println("Write error:", err)
		}
	})

	return router
}
