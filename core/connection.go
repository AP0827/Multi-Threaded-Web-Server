package core

import (
	mtwshttp "MTWS/http"
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

type ConnectionOptions struct {
	ReadTimeout              time.Duration
	WriteTimeout             time.Duration
	IdleTimeout              time.Duration
	MaxRequestsPerConnection int
	DisableKeepAlive         bool
	Metrics                  *Metrics
}

func HandleConnection(conn net.Conn, router *Router) {
	HandleConnectionWithOptions(conn, router, ConnectionOptions{})
}

func HandleConnectionWithOptions(conn net.Conn, router *Router, options ConnectionOptions) {
	defer conn.Close()

	if options.ReadTimeout <= 0 {
		options.ReadTimeout = 5 * time.Second
	}
	if options.WriteTimeout <= 0 {
		options.WriteTimeout = 5 * time.Second
	}
	if options.IdleTimeout <= 0 {
		options.IdleTimeout = 30 * time.Second
	}
	if options.MaxRequestsPerConnection <= 0 {
		options.MaxRequestsPerConnection = 100
	}

	started := time.Now()
	options.Metrics.IncActiveConnection()
	defer options.Metrics.DecActiveConnection()

	log.Println("New request from", conn.RemoteAddr())

	reader := bufio.NewReader(conn)

	for served := 0; ; served++ {
		readTimeout := options.ReadTimeout
		if served > 0 {
			readTimeout = options.IdleTimeout
		}

		// Prevent slowloris-style hanging on the first request and idle keep-alive hoarding later.
		if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			log.Println("Failed to set read deadline:", err)
			return
		}

		req, err := mtwshttp.ParseRequest(reader)
		if err != nil {
			if errors.Is(err, io.EOF) && served > 0 {
				return
			}

			var securityErr *mtwshttp.SecurityError
			if errors.As(err, &securityErr) {
				options.Metrics.IncWAFBlock()
				logSecurityEvent(conn, securityErr.Field, securityErr.Pattern, securityErr.RuleID, StatusForbidden.Code)
				writeErrorResponseWithOptions(conn, "HTTP/1.1", StatusForbidden, ResponseOptions{
					WriteTimeout: options.WriteTimeout,
					Metrics:      options.Metrics,
				})
				return
			}

			options.Metrics.IncParseReject()
			logParseReject(conn, err.Error(), StatusBadRequest.Code)
			writeErrorResponseWithOptions(conn, "HTTP/1.1", StatusBadRequest, ResponseOptions{
				WriteTimeout: options.WriteTimeout,
				Metrics:      options.Metrics,
			})
			return
		}
		options.Metrics.IncRequest()

		if router == nil {
			log.Println("Router is not configured")
			writeErrorResponseWithOptions(conn, req.Version(), StatusInternalServerError, ResponseOptions{
				WriteTimeout: options.WriteTimeout,
				Metrics:      options.Metrics,
			})
			return
		}

		remainingRequests := options.MaxRequestsPerConnection - served - 1
		keepAlive := shouldKeepAlive(req, options, remainingRequests)
		log.Printf("Request method=%s path=%s version=%s keep_alive=%t", req.Method(), req.Path(), req.Version(), keepAlive)

		writer := NewResponseWriterWithOptions(conn, req.Version(), ResponseOptions{
			WriteTimeout:         options.WriteTimeout,
			Metrics:              options.Metrics,
			KeepAlive:            keepAlive,
			KeepAliveTimeout:     options.IdleTimeout,
			KeepAliveMaxRequests: remainingRequests,
		})
		router.ServeHTTP(writer, req)
		if !writer.Wrote() {
			_ = writer.WriteText(StatusInternalServerError, StatusInternalServerError.Body)
			keepAlive = false
		}
		logAccessEvent(conn, req, writer.StatusCode(), writer.BytesWritten(), time.Since(started))
		if !keepAlive {
			return
		}
	}
}

func shouldKeepAlive(req *mtwshttp.Request, options ConnectionOptions, remainingRequests int) bool {
	if options.DisableKeepAlive || remainingRequests <= 0 {
		return false
	}

	connection := strings.ToLower(req.Headers()["Connection"])
	for _, token := range strings.Split(connection, ",") {
		if strings.TrimSpace(token) == "close" {
			return false
		}
	}
	return true
}
