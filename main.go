package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Handler struct {
	client http.Client
	upstream string
}

func  New(upstream string) *Handler{
	_, port, err := net.SplitHostPort(upstream)
	if err != nil && !strings.HasSuffix(err.Error(), "missing port in address"){
		panic(err)
	}
	if len(port) == 0{
		upstream = upstream + ":80"
	}
	return  &Handler{
		client: http.Client{
			Timeout:       30*time.Second,
		},
		upstream: upstream,
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
	req := &http.Request{Method:r.Method, URL: URL}
	req.Header = make(map[string][]string)

	for s := range r.Header {
		if len(s) == 0 {
			continue
		}
		v := r.Header.Get(s)
		if len(v) == 0 {
			continue
		}

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

	w.WriteHeader(resp.StatusCode)


	p := make([]byte, 4)
	for {
		read, err := resp.Body.Read(p)
		if err == io.EOF || read == 0{
			break
		}

		written, err := w.Write(p)
		if err == io.EOF || written == 0{
			break
		}

	}
}

func main()  {
	upstream := flag.String("upstream", "", "HTTP upstream, e.g. 192.168.3.1:81 or just 192.168.3.1")
	bind := flag.String("bind", "0.0.0.0:8000", "Bind addr, e.g. 0.0.0.0:8000")
	flag.Parse()
	if *upstream == "" {
		fmt.Println("upstream cannot be empty")
		os.Exit(1)
	}
	handler := New(*upstream)

	srv := http.Server{
		Addr: *bind,
		Handler: handler,
		ReadTimeout: 30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic("Listen failed")
	}

}
