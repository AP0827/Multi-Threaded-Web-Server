package core

import (
	mtwshttp "MTWS/http"
	"bufio"
	"errors"
	"log"
	"net"
	"time"
)

func HandleConnection(conn net.Conn, router *Router) {
	defer conn.Close()

	log.Println("New request from", conn.RemoteAddr())

	// Prevent slowloris-style hanging
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.Println("Failed to set read deadline:", err)
		return
	}

	reader := bufio.NewReader(conn)

	req, err := mtwshttp.ParseRequest(reader)
	if err != nil {
		var securityErr *mtwshttp.SecurityError
		if errors.As(err, &securityErr) {
			log.Printf("WAF blocked request from %s: field=%s pattern=%q", conn.RemoteAddr(), securityErr.Field, securityErr.Pattern)
			writeErrorResponse(conn, "HTTP/1.1", StatusForbidden)
			return
		}

		log.Println("Parse error:", err)
		writeErrorResponse(conn, "HTTP/1.1", StatusBadRequest)
		return
	}

	if router == nil {
		log.Println("Router is not configured")
		writeErrorResponse(conn, req.Version(), StatusInternalServerError)
		return
	}

	log.Printf("Request method=%s path=%s version=%s", req.Method(), req.Path(), req.Version())

	writer := NewResponseWriter(conn, req.Version())
	router.ServeHTTP(writer, req)
}
