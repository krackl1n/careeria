package redis

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
)

func buildTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: strings.TrimSpace(cfg.ServerName),
	}

	if cfg.CAFile != "" {
		rootCAs, err := loadRootCAs(cfg.CAFile)
		if err != nil {
			return nil, err
		}

		tlsConfig.RootCAs = rootCAs
	}

	if cfg.CertFile != "" {
		certificate, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load redis tls client certificate: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{
			certificate,
		}
	}

	return tlsConfig, nil
}

func loadRootCAs(path string) (*x509.CertPool, error) {
	caPEM, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read redis tls ca file: %w", err)
	}

	rootCAs, err := x509.SystemCertPool()
	if err != nil || rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	if ok := rootCAs.AppendCertsFromPEM(caPEM); !ok {
		return nil, errors.New("redis tls ca file contains no valid certificates")
	}

	return rootCAs, nil
}
