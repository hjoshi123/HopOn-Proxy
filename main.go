package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	zap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	var (
		flagCertPath = flag.String("cert", "", "/Users/hemantj/Documents/scratch/proxy/cert.pem")
		flagKeyPath  = flag.String("key", "", "/Users/hemantj/Documents/scratch/proxy/key.pem")
		flagAvoid    = flag.String("avoid", "", "Site to be avoided")
		flagVerbose  = flag.Bool("verbose", true, "Set log level to DEBUG")

		flagDestDialTimeout         = flag.Duration("destdialtimeout", 10*time.Second, "Destination dial timeout")
		flagDestReadTimeout         = flag.Duration("destreadtimeout", 5*time.Second, "Destination read timeout")
		flagDestWriteTimeout        = flag.Duration("destwritetimeout", 5*time.Second, "Destination write timeout")
		flagClientReadTimeout       = flag.Duration("clientreadtimeout", 5*time.Second, "Client read timeout")
		flagClientWriteTimeout      = flag.Duration("clientwritetimeout", 5*time.Second, "Client write timeout")
		flagServerReadTimeout       = flag.Duration("serverreadtimeout", 30*time.Second, "Server read timeout")
		flagServerReadHeaderTimeout = flag.Duration("serverreadheadertimeout", 30*time.Second, "Server read header timeout")
		flagServerWriteTimeout      = flag.Duration("serverwritetimeout", 30*time.Second, "Server write timeout")
		flagServerIdleTimeout       = flag.Duration("serveridletimeout", 30*time.Second, "Server idle timeout")

		flagLetsEncrypt = flag.Bool("letsencrypt", false, "Use letsencrypt for https")
		flagLEWhitelist = flag.String("lewhitelist", "", "Hostname to whitelist for letsencrypt")
		flagLECacheDir  = flag.String("lecachedir", "/tmp", "Cache directory for certificates")
	)

	flag.Parse()

	c := zap.NewProductionConfig()
	c.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if *flagVerbose {
		c.Level.SetLevel(zapcore.DebugLevel)
	} else {
		c.Level.SetLevel(zapcore.ErrorLevel)
	}

	logger, err := c.Build()
	if err != nil {
		log.Fatalln("Error: failed to initiate logger")
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Fatalln("Cannot sync log")
		}
	}()

	stdLogger := zap.NewStdLog(logger)

	p := &Proxy{
		ForwardHTTPProxy:   NewForwardHTTPProxy(stdLogger),
		Logger:             logger,
		DestDialTimeout:    *flagDestDialTimeout,
		DestReadTimeout:    *flagDestReadTimeout,
		DestWriteTimeout:   *flagDestWriteTimeout,
		ClientReadTimeout:  *flagClientReadTimeout,
		ClientWriteTimeout: *flagClientWriteTimeout,
		Avoid:              *flagAvoid,
	}

	s := &http.Server{
		Addr:              ":8080",
		Handler:           p,
		ReadTimeout:       *flagServerReadTimeout,
		ReadHeaderTimeout: *flagServerReadHeaderTimeout,
		WriteTimeout:      *flagServerWriteTimeout,
		IdleTimeout:       *flagServerIdleTimeout,
		TLSNextProto:      map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}

	if *flagLetsEncrypt {
		if *flagLEWhitelist == "" {
			p.Logger.Fatal("error no le whitelist")
		}

		if *flagLECacheDir == "/tmp" {
			p.Logger.Info("cache dir set to /tmp")
		}

		m := &autocert.Manager{
			Cache:      autocert.DirCache(*flagLECacheDir),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(*flagLEWhitelist),
		}

		s.Addr = ":https"
		s.TLSConfig = m.TLSConfig()
	}

	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		p.Logger.Info("Server shutting down")
		if err = s.Shutdown(context.Background()); err != nil {
			p.Logger.Error("Server shutdown failed", zap.Error(err))
		}
		close(idleConnsClosed)
	}()

	p.Logger.Info("Server starting", zap.String("Address", s.Addr))

	var svrErr error
	if *flagCertPath != "" && *flagKeyPath != "" || *flagLetsEncrypt {
		svrErr = s.ListenAndServeTLS(*flagCertPath, *flagKeyPath)
	} else {
		svrErr = s.ListenAndServe()
	}

	if svrErr != http.ErrServerClosed {
		p.Logger.Error("Listening for incoming connections failed", zap.Error(svrErr))
	}

	<-idleConnsClosed
	p.Logger.Info("Total data", zap.Int64("Lol", Sum))
	p.Logger.Info("Server stopped")
}
