package main

import (
	"MTWS/config"
	"MTWS/core"
	mtwshttp "MTWS/http"
	"MTWS/pool"
	"MTWS/security/ratelimiter"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"mime"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cfg config.Config) error {
	if cfg.TLSPartiallyConfigured() {
		return fmt.Errorf("TLS requires both MTWS_TLS_CERT_FILE and MTWS_TLS_KEY_FILE")
	}

	l, err := listen(cfg)
	if err != nil {
		return err
	}
	defer l.Close()

	// The bounded queue is part of the overload policy; saturated queues fail closed with 503.
	jobs := make(chan pool.Job, cfg.JobQueueSize)
	workers := pool.StartWorkerPool(cfg.WorkerPoolSize, jobs)
	metrics := core.NewMetrics()
	router := buildRouterWithMetrics(metrics, cfg.StaticDir)
	limiter := ratelimiter.New(cfg.RateLimitRate, cfg.RateLimitCapacity)

	logStartup(cfg)

	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	serveErr := serve(ctx, l, jobs, router, limiter, cfg, metrics)
	stopped := make(chan struct{})
	go func() {
		workers.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(cfg.ShutdownTimeout):
		return fmt.Errorf("shutdown timed out after %s", cfg.ShutdownTimeout)
	}

	return serveErr
}

func serve(ctx context.Context, l net.Listener, jobs chan<- pool.Job, router *core.Router, limiter *ratelimiter.Limiter, cfg config.Config, metrics *core.Metrics) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				return nil
			}
			log.Println("Accept error:", err)
			continue
		}
		metrics.IncAcceptedConnection()

		remoteAddr := conn.RemoteAddr().String()
		clientIP, _, err := net.SplitHostPort(remoteAddr)
		if err != nil {
			clientIP = remoteAddr
		}

		if !cfg.RateLimitEnabled || limiter.Allow(clientIP) {
			job := pool.Job{
				Conn:   conn,
				Router: router,
				Options: core.ConnectionOptions{
					ReadTimeout:              cfg.ReadTimeout,
					WriteTimeout:             cfg.WriteTimeout,
					IdleTimeout:              cfg.IdleTimeout,
					MaxRequestsPerConnection: cfg.MaxKeepAlive,
					Metrics:                  metrics,
				},
			}
			if enqueueJob(ctx, jobs, job, cfg.QueueTimeout) {
				continue
			}

			metrics.IncQueueReject()
			log.Printf("Status 503: worker queue saturated for %s", clientIP)
			if err := core.NewResponseWriterWithOptions(conn, "HTTP/1.1", core.ResponseOptions{
				WriteTimeout: cfg.WriteTimeout,
				Metrics:      metrics,
			}).WriteText(core.StatusServiceUnavailable, ""); err != nil {
				log.Println("Write error:", err)
			}
			conn.Close()
			continue
		}

		metrics.IncRateLimited()
		log.Printf("Status 429: request blocked by token bucket policy for %s", clientIP)
		if err := core.NewResponseWriterWithOptions(conn, "HTTP/1.1", core.ResponseOptions{
			WriteTimeout: cfg.WriteTimeout,
			Metrics:      metrics,
		}).WriteText(core.StatusTooManyRequests, ""); err != nil {
			log.Println("Write error:", err)
		}
		conn.Close()
	}
}

func listen(cfg config.Config) (net.Listener, error) {
	if !cfg.TLSEnabled() {
		return net.Listen("tcp", cfg.ServerAddress)
	}

	cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load TLS certificate: %w", err)
	}

	return tls.Listen("tcp", cfg.ServerAddress, &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	})
}

func enqueueJob(ctx context.Context, jobs chan<- pool.Job, job pool.Job, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case jobs <- job:
		return true
	case <-timer.C:
		return false
	case <-ctx.Done():
		job.Conn.Close()
		return true
	}
}

func logStartup(cfg config.Config) {
	log.Printf("MTWS listening on %s", cfg.ServerAddress)
	log.Printf("Workers=%d queue=%d read_timeout=%s write_timeout=%s idle_timeout=%s max_keepalive=%d queue_timeout=%s shutdown_timeout=%s",
		cfg.WorkerPoolSize,
		cfg.JobQueueSize,
		cfg.ReadTimeout,
		cfg.WriteTimeout,
		cfg.IdleTimeout,
		cfg.MaxKeepAlive,
		cfg.QueueTimeout,
		cfg.ShutdownTimeout,
	)
	if cfg.TLSEnabled() {
		log.Println("TLS enabled")
	} else {
		log.Println("TLS disabled; serving plaintext HTTP")
	}
	if cfg.RateLimitEnabled {
		log.Printf("Rate limiter enabled: rate=%.2f capacity=%.2f", cfg.RateLimitRate, cfg.RateLimitCapacity)
	} else {
		log.Println("Rate limiter disabled by environment")
	}
}

func buildRouter() *core.Router {
	return buildRouterWithMetrics(core.NewMetrics(), config.DefaultStaticDir)
}

func buildRouterWithMetrics(metrics *core.Metrics, staticDir string) *core.Router {
	router := core.NewRouter()

	router.Handle("/", func(w *core.ResponseWriter, req *mtwshttp.Request) {
		if !requireMethod(w, req, "GET") {
			return
		}
		body := "MTWS baseline server is running\n"
		if err := w.WriteText(core.StatusOK, body); err != nil {
			log.Println("Write error:", err)
		}
	})

	router.Handle("/health", func(w *core.ResponseWriter, req *mtwshttp.Request) {
		if !requireMethod(w, req, "GET") {
			return
		}
		body := "ok\n"
		if err := w.WriteText(core.StatusOK, body); err != nil {
			log.Println("Write error:", err)
		}
	})

	router.Handle("/ready", func(w *core.ResponseWriter, req *mtwshttp.Request) {
		if !requireMethod(w, req, "GET") {
			return
		}
		if err := w.WriteText(core.StatusOK, "ready\n"); err != nil {
			log.Println("Write error:", err)
		}
	})

	router.Handle("/metrics", func(w *core.ResponseWriter, req *mtwshttp.Request) {
		if !requireMethod(w, req, "GET") {
			return
		}
		if err := w.WriteText(core.StatusOK, metrics.Prometheus()); err != nil {
			log.Println("Write error:", err)
		}
	})

	router.Handle("/submit", func(w *core.ResponseWriter, req *mtwshttp.Request) {
		if !requireMethod(w, req, "POST") {
			return
		}
		body := "MTWS baseline server is running\n"
		if err := w.WriteText(core.StatusOK, body); err != nil {
			log.Println("Write error:", err)
		}
	})

	router.Handle("/search", func(w *core.ResponseWriter, req *mtwshttp.Request) {
		if !requireMethod(w, req, "GET") {
			return
		}
		_, query, _ := strings.Cut(req.Path(), "?")
		body := fmt.Sprintf("query=%s\n", query)
		if err := w.WriteText(core.StatusOK, body); err != nil {
			log.Println("Write error:", err)
		}
	})

	router.HandlePrefix("/static/", secureStaticHandler(staticDir))

	return router
}

func requireMethod(w *core.ResponseWriter, req *mtwshttp.Request, method string) bool {
	if req.Method() == method {
		return true
	}
	if err := w.WriteText(core.StatusMethodNotAllowed, ""); err != nil {
		log.Println("Write error:", err)
	}
	return false
}

func secureStaticHandler(root string) core.HandlerFunc {
	absRoot, rootErr := resolveStaticRoot(root)

	return func(w *core.ResponseWriter, req *mtwshttp.Request) {
		if !requireMethod(w, req, "GET") {
			return
		}

		pathOnly, _, _ := strings.Cut(req.Path(), "?")
		rawRelative := strings.TrimPrefix(pathOnly, "/static/")
		if rawRelative == "" {
			rawRelative = "index.html"
		}

		decodedRelative, err := url.PathUnescape(rawRelative)
		if err != nil || !isSafeStaticPath(decodedRelative) {
			_ = w.WriteText(core.StatusBadRequest, "")
			return
		}
		if rootErr != nil {
			log.Printf("Static root error: %v", rootErr)
			_ = w.WriteText(core.StatusInternalServerError, "")
			return
		}

		candidate := filepath.Join(absRoot, filepath.FromSlash(decodedRelative))
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				_ = w.WriteText(core.StatusNotFound, "")
				return
			}
			log.Printf("Static file resolution error: %v", err)
			_ = w.WriteText(core.StatusInternalServerError, "")
			return
		}
		if !isWithinRoot(absRoot, resolved) {
			_ = w.WriteText(core.StatusForbidden, "")
			return
		}

		info, err := os.Stat(resolved)
		if err != nil {
			if os.IsNotExist(err) {
				_ = w.WriteText(core.StatusNotFound, "")
				return
			}
			log.Printf("Static file stat error: %v", err)
			_ = w.WriteText(core.StatusInternalServerError, "")
			return
		}
		if info.IsDir() {
			resolved, err = filepath.EvalSymlinks(filepath.Join(resolved, "index.html"))
			if err != nil {
				if os.IsNotExist(err) {
					_ = w.WriteText(core.StatusNotFound, "")
					return
				}
				log.Printf("Static index resolution error: %v", err)
				_ = w.WriteText(core.StatusInternalServerError, "")
				return
			}
			if !isWithinRoot(absRoot, resolved) {
				_ = w.WriteText(core.StatusForbidden, "")
				return
			}
			info, err = os.Stat(resolved)
			if err != nil || info.IsDir() {
				_ = w.WriteText(core.StatusNotFound, "")
				return
			}
		}

		body, err := os.ReadFile(resolved)
		if err != nil {
			log.Printf("Static file read error: %v", err)
			_ = w.WriteText(core.StatusInternalServerError, "")
			return
		}

		contentType := mime.TypeByExtension(filepath.Ext(resolved))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		if err := w.WriteBytes(core.StatusOK, contentType, body); err != nil {
			log.Println("Write error:", err)
		}
	}
}

func resolveStaticRoot(root string) (string, error) {
	if filepath.IsAbs(root) {
		return filepath.EvalSymlinks(root)
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for i := 0; i < 6; i++ {
		candidate := filepath.Join(wd, root)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return filepath.EvalSymlinks(candidate)
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absRoot)
}

func isSafeStaticPath(path string) bool {
	if path == "" {
		return true
	}
	if strings.Contains(path, "\\") || strings.Contains(path, ":") {
		return false
	}
	for _, r := range path {
		if r <= 31 || r == 127 {
			return false
		}
	}

	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if cleaned == "." {
		return true
	}
	return !strings.HasPrefix(cleaned, "../") && cleaned != ".." && !strings.HasPrefix(cleaned, "/")
}

func isWithinRoot(root string, target string) bool {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return relative == "." || (!strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != ".." && !filepath.IsAbs(relative))
}
