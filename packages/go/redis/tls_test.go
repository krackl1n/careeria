package redis

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildTLSConfigDisabled(t *testing.T) {
	t.Parallel()

	cfg, err := buildTLSConfig(TLSConfig{ServerName: "ignored.example"})

	require.NoError(t, err)
	require.Nil(t, cfg)
}

func TestBuildTLSConfig(t *testing.T) {
	t.Parallel()

	certFile, keyFile := writeTestCertificate(t)

	cfg, err := buildTLSConfig(TLSConfig{
		Enabled:    true,
		ServerName: " redis.example ",
		CAFile:     certFile,
		CertFile:   certFile,
		KeyFile:    keyFile,
	})

	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
	require.Equal(t, "redis.example", cfg.ServerName)
	require.NotNil(t, cfg.RootCAs)
	require.Len(t, cfg.Certificates, 1)
	require.NotEmpty(t, cfg.Certificates[0].Certificate)
}

func TestBuildTLSConfigWithoutOptionalFiles(t *testing.T) {
	t.Parallel()

	cfg, err := buildTLSConfig(TLSConfig{Enabled: true})

	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Nil(t, cfg.RootCAs)
	require.Empty(t, cfg.Certificates)
}

func TestBuildTLSConfigErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	invalidCAFile := filepath.Join(dir, "invalid-ca.pem")
	require.NoError(t, os.WriteFile(invalidCAFile, []byte("not a certificate"), 0o600))

	invalidCertFile := filepath.Join(dir, "invalid-client.pem")
	require.NoError(t, os.WriteFile(invalidCertFile, []byte("not a certificate"), 0o600))

	tests := []struct {
		name        string
		cfg         TLSConfig
		wantErrPart string
	}{
		{
			name: "missing ca file",
			cfg: TLSConfig{
				Enabled: true,
				CAFile:  filepath.Join(dir, "missing-ca.pem"),
			},
			wantErrPart: "read redis tls ca file",
		},
		{
			name: "invalid ca file",
			cfg: TLSConfig{
				Enabled: true,
				CAFile:  invalidCAFile,
			},
			wantErrPart: "redis tls ca file contains no valid certificates",
		},
		{
			name: "invalid client certificate",
			cfg: TLSConfig{
				Enabled:  true,
				CertFile: invalidCertFile,
				KeyFile:  invalidCertFile,
			},
			wantErrPart: "load redis tls client certificate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := buildTLSConfig(tt.cfg)

			require.Nil(t, cfg)
			require.ErrorContains(t, err, tt.wantErrPart)
		})
	}
}

func TestLoadRootCAs(t *testing.T) {
	t.Parallel()

	certFile, _ := writeTestCertificate(t)
	certPEM, err := os.ReadFile(certFile)
	require.NoError(t, err)
	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	pool, err := loadRootCAs(certFile)

	require.NoError(t, err)
	require.NotNil(t, pool)
	_, err = cert.Verify(x509.VerifyOptions{
		Roots:   pool,
		DNSName: "redis.example",
	})
	require.NoError(t, err)
}

func writeTestCertificate(t *testing.T) (string, string) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "redis.example"},
		DNSNames:              []string{"redis.example"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	require.NoError(t, err)
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	dir := t.TempDir()
	certFile := filepath.Join(dir, "redis.pem")
	keyFile := filepath.Join(dir, "redis-key.pem")
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDER,
	}), 0o600))

	return certFile, keyFile
}
