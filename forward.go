package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	realip "github.com/tomasen/realip"
	"go.uber.org/zap"
)

// Sum is the total amount of data consumed
var Sum int64

// Proxy is a HTTPS Forward proxy
type Proxy struct {
	Logger             *zap.Logger
	Avoid              string
	ForwardHTTPProxy   *httputil.ReverseProxy
	DestDialTimeout    time.Duration
	DestReadTimeout    time.Duration
	DestWriteTimeout   time.Duration
	ClientReadTimeout  time.Duration
	ClientWriteTimeout time.Duration
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.Logger.Info("Incoming request", zap.String("host", r.Host))

	// continue with replication i.e simple forwarding if the traffic is http
	// else use tunneling to CONNECT https data since TLS doesnt allow
	// modification of data
	if r.URL.Scheme == "http" {
		p.handleHTTP(w, r)
	} else {
		p.handleTunneling(w, r)
	}
}

func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	p.Logger.Debug("Got HTTP request", zap.String("host", r.Host))
	clientIP := realip.FromRequest(r)
	if p.Avoid != "" && strings.Contains(r.Host, p.Avoid) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusMethodNotAllowed)
		return
	}
	p.Logger.Debug("HTTP Request IP", zap.String("IP", clientIP))
	p.ForwardHTTPProxy.ServeHTTP(w, r)
}

func (p *Proxy) handleTunneling(w http.ResponseWriter, r *http.Request) {
	if p.Avoid != "" && strings.Contains(r.Host, p.Avoid) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusMethodNotAllowed)
		return
	}

	// verifies if the first request method is CONNECT
	if r.Method != http.MethodConnect {
		p.Logger.Info("Method not allowed", zap.String("method", r.Method))
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	clientIP := realip.FromRequest(r)
	p.Logger.Debug("Connecting", zap.String("host", r.Host))
	p.Logger.Debug("HTTPS Request IP", zap.String("IP", clientIP))

	// dials or creates a TCP connection to the destination
	destConn, err := net.DialTimeout("tcp", r.Host, p.DestDialTimeout)
	if err != nil {
		p.Logger.Error("Destination dial failed", zap.Error(err))
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// p.Logger.Debug("Connected", zap.String("host", r.Host))
	w.WriteHeader(http.StatusOK)

	p.Logger.Debug("Hijacking", zap.String("host", r.Host))

	// hijacks the established connection from server to http handler
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		p.Logger.Error("Hijacking not supported")
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		p.Logger.Error("Hijacking failed", zap.Error(err))
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	now := time.Now()
	if clientConn.SetReadDeadline(now.Add(p.ClientReadTimeout)) != nil {
		p.Logger.Error("Error in setting read deadline")
	}
	if clientConn.SetWriteDeadline(now.Add(p.ClientWriteTimeout)) != nil {
		p.Logger.Error("Error in setting write deadline")
	}
	if destConn.SetReadDeadline(now.Add(p.DestReadTimeout)) != nil {
		p.Logger.Error("Error in setting destination read deadline")
	}
	if destConn.SetWriteDeadline(now.Add(p.DestWriteTimeout)) != nil {
		p.Logger.Error("Error in setting destination write deadline")
	}

	// streams data between client and destination
	// Go routines stream bidirectionally by spawning two stream copiers
	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)
}

func transfer(dest io.WriteCloser, src io.ReadCloser) {
	defer func() { _ = dest.Close() }()
	defer func() { _ = src.Close() }()
	nr, err := io.Copy(dest, src)
	if err != nil {
		fmt.Println("Error in Copying")
	}
	Sum += nr
}

// NewForwardHTTPProxy returns a reverse proxy that take incoming request and
// sends it to another server
func NewForwardHTTPProxy(logger *log.Logger) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "")
		}
	}

	return &httputil.ReverseProxy{
		ErrorLog: logger,
		Director: director,
	}
}
