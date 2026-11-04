package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
)

func newTransportCredentials(cfg TLSConfig) (credentials.TransportCredentials, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: cfg.ServerName,
	}

	if cfg.CAFile != "" {
		rawCA, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read ca file: %w", err)
		}

		pool := x509.NewCertPool()
		if ok := pool.AppendCertsFromPEM(rawCA); !ok {
			return nil, fmt.Errorf("append ca cert")
		}

		tlsConfig.RootCAs = pool
	}

	if cfg.ClientCertFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCertFile, cfg.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client certificate: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return credentials.NewTLS(tlsConfig), nil
}
