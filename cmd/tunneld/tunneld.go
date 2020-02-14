// Copyright (C) 2017 Michał Matczuk
// Use of this source code is governed by an AGPL-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"golang.org/x/net/http2"

	"github.com/virgild/go-http-tunnel"
	"github.com/virgild/go-http-tunnel/log"
)

func main() {
	opts := parseArgs()

	if opts.version {
		fmt.Println(version)
		return
	}

	fmt.Println(banner)

	// Configure logger
	logger := log.NewStackdriverLogger(&log.StackdriverLoggerOptions{
		LogLevel:    opts.logLevel,
		ServiceName: "tunneld",
		Version:     version,
	})

	tlsconf, err := tlsConfig(opts)
	if err != nil {
		fatal("failed to configure tls: %s", err)
	}

	// setup server
	server, err := tunnel.NewServer(&tunnel.ServerConfig{
		Addr:      opts.tunnelAddr,
		SNIAddr:   opts.sniAddr,
		TLSConfig: tlsconf,
		Logger:    logger,
	})
	if err != nil {
		fatal("failed to create server: %s", err)
	}

	// start HTTP
	if opts.httpAddr != "" {
		go func() {
			logger.Log(
				"level", 1,
				"action", "start http",
				"addr", opts.httpAddr,
			)

			fatal("failed to start HTTP: %s", http.ListenAndServe(opts.httpAddr, server))
		}()
	}

	// Load cert for https and API server:
	cert, err := loadX509KeyPair(context.Background(), opts.tlsCrt, opts.tlsKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load certificate: %s", err.Error())
		os.Exit(1)
	}

	// start HTTPS
	if opts.httpsAddr != "" {
		go func() {
			logger.Log(
				"level", 1,
				"action", "start https",
				"addr", opts.httpsAddr,
			)

			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{cert},
			}

			s := &http.Server{
				Addr:      opts.httpsAddr,
				Handler:   server,
				TLSConfig: tlsConfig,
			}
			err = http2.ConfigureServer(s, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ConfigureServer error: %s", err)
				os.Exit(1)
			}

			fatal("failed to start HTTPS: %s", s.ListenAndServeTLS("", ""))
		}()
	}

	// start API server
	if opts.apiAddr != "" {
		go func() {
			logger.Log(
				"level", 1,
				"action", "start API server",
				"addr", opts.apiAddr,
			)

			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{cert},
			}

			s := &http.Server{
				Addr:      opts.apiAddr,
				Handler:   server.APIHandler(),
				TLSConfig: tlsConfig,
			}
			err := http2.ConfigureServer(s, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ConfigureServer error: %s", err)
				os.Exit(1)
			}

			fatal("failed to start API server: %s", s.ListenAndServeTLS("", ""))
		}()
	}

	server.Start()
}

func tlsConfig(opts *options) (*tls.Config, error) {
	// load certs
	cert, err := loadX509KeyPair(context.Background(), opts.tlsCrt, opts.tlsKey)
	if err != nil {
		return nil, err
	}

	// load root CA for client authentication
	clientAuth := tls.RequireAnyClientCert
	var roots *x509.CertPool
	if opts.rootCA != "" {
		roots = x509.NewCertPool()
		var rootPEM []byte

		if strings.HasPrefix(opts.rootCA, "gs://") {
			gcsPath := strings.TrimPrefix(opts.rootCA, "gs://")
			nameElements := strings.SplitN(gcsPath, "/", 2)
			bucketName := nameElements[0]
			filePath := nameElements[1]
			rootPEM, err = loadGCSFile(context.Background(), bucketName, filePath)
			if err != nil {
				return nil, err
			}
		} else {
			rootPEM, err = ioutil.ReadFile(opts.rootCA)
			if err != nil {
				return nil, err
			}
		}
		if ok := roots.AppendCertsFromPEM(rootPEM); !ok {
			return nil, err
		}
		clientAuth = tls.RequireAndVerifyClientCert
	}

	return &tls.Config{
		Certificates:           []tls.Certificate{cert},
		ClientAuth:             clientAuth,
		ClientCAs:              roots,
		SessionTicketsDisabled: true,
		MinVersion:             tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		PreferServerCipherSuites: true,
		NextProtos:               []string{"h2"},
	}, nil
}

func fatal(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Fprint(os.Stderr, "\n")
	os.Exit(1)
}

func loadX509KeyPair(ctx context.Context, certFile, keyFile string) (tls.Certificate, error) {
	var certPEMBlock, keyPEMBlock []byte
	var err error

	if strings.HasPrefix(certFile, "gs://") {
		gcsPath := strings.TrimPrefix(certFile, "gs://")
		nameElements := strings.SplitN(gcsPath, "/", 2)
		bucketName := nameElements[0]
		filePath := nameElements[1]
		certPEMBlock, err = loadGCSFile(ctx, bucketName, filePath)
		if err != nil {
			return tls.Certificate{}, err
		}
	} else {
		certPEMBlock, err = ioutil.ReadFile(certFile)
		if err != nil {
			return tls.Certificate{}, err
		}
	}

	if strings.HasPrefix(keyFile, "gs://") {
		gcsPath := strings.TrimPrefix(keyFile, "gs://")
		nameElements := strings.SplitN(gcsPath, "/", 2)
		bucketName := nameElements[0]
		filePath := nameElements[1]
		keyPEMBlock, err = loadGCSFile(ctx, bucketName, filePath)
		if err != nil {
			return tls.Certificate{}, err
		}
	} else {
		keyPEMBlock, err = ioutil.ReadFile(keyFile)
		if err != nil {
			return tls.Certificate{}, err
		}
	}

	return tls.X509KeyPair(certPEMBlock, keyPEMBlock)
}

func loadGCSFile(ctx context.Context, bucketName, path string) ([]byte, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	bucket := client.Bucket(bucketName)
	obj := bucket.Object(path)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}

	block, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return block, nil
}
