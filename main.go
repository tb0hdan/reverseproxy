package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Handler - HTTP handler with bound methods
type Handler struct {
	client   http.Client
	upstream string
}

// New - create new HTTP handler
func New(upstream string) *Handler {
	_, port, err := net.SplitHostPort(upstream)
	if err != nil && !strings.HasSuffix(err.Error(), "missing port in address") {
		panic(err)
	}

	if len(port) == 0 {
		upstream += ":80"
	}

	return &Handler{
		client: http.Client{
			Timeout: 30 * time.Second,
		},
		upstream: upstream,
	}
}

// ContentWriter - straightforward io to io copy
func ContentWriter(src io.Reader, dst io.Writer) {
	p := make([]byte, 4)

	for {
		read, err := src.Read(p)
		if err == io.EOF || read == 0 {
			break
		}

		written, err := dst.Write(p)
		if err == io.EOF || written == 0 {
			break
		}
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		requestURI string
	)

	parsed, err := url.Parse(r.Referer())

	if err != nil {
		panic(err)
	}

	if strings.HasSuffix(parsed.Path, "/") {
		requestURI = r.RequestURI
	} else {
		requestURI = parsed.Path + r.RequestURI
	}

	remoteURL := fmt.Sprintf("http://%s%s", h.upstream, requestURI)

	fmt.Println(r.RemoteAddr, remoteURL)

	URL, err := url.Parse(remoteURL)
	if err != nil {
		panic(err)
	}

	req := &http.Request{Method: r.Method, URL: URL}
	req.Header = make(map[string][]string)

	// Copy request headers
	for s := range r.Header {
		v := r.Header.Get(s)

		// Skip accept-encoding as we don't support gzip yet
		// gzip.NewReader(r)
		if s == "Accept-Encoding" {
			continue
		}

		req.Header.Set(s, v)
	}

	resp, err := h.client.Do(req)

	if err != nil {
		panic(err)
	} else {
		defer resp.Body.Close()
	}

	// Copy response headers
	for s := range resp.Header {
		v := resp.Header.Get(s)
		w.Header().Set(s, v)
	}

	ContentWriter(resp.Body, w)
}

func main() {
	upstream := flag.String("upstream", "", "HTTP upstream, e.g. 192.168.3.1:81 or just 192.168.3.1")
	bind := flag.String("bind", "0.0.0.0:8000", "Bind addr, e.g. 0.0.0.0:8000")
	flag.Parse()

	if *upstream == "" {
		fmt.Println("upstream cannot be empty")
		os.Exit(1)
	}

	handler := New(*upstream)

	srv := http.Server{
		Addr:         *bind,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(fmt.Sprintf("Listen failed with: %v\n", err))
		}
	}()
	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)

	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		panic(fmt.Sprintf("Couldn't perform shutdown:%+v", err))
	}
}
