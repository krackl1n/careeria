package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/IBM/sarama"
)

func (c TLSConfig) validate() error {
	if !c.Enabled {
		return nil
	}

	if (c.CertFile == "") != (c.KeyFile == "") {
		return errors.New(
			"kafka: tls cert file and key file must be configured together",
		)
	}

	if !c.InsecureSkipVerify && strings.TrimSpace(c.ServerName) == "" {
		return errors.New(
			"kafka: tls server name is required when certificate verification is enabled",
		)
	}

	return nil
}

func (c SASLConfig) validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Username == "" {
		return errors.New("kafka: sasl username is required")
	}

	if c.Password == "" {
		return errors.New("kafka: sasl password is required")
	}

	switch c.Mechanism {
	case "",
		SASLMechanismPlain,
		SASLMechanismSCRAMSHA256,
		SASLMechanismSCRAMSHA512:
		return nil
	default:
		return fmt.Errorf(
			"kafka: unsupported sasl mechanism %q",
			c.Mechanism,
		)
	}
}

func buildTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	config := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         cfg.ServerName,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	if cfg.CAFile != "" {
		caPEM, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("kafka: read tls ca file: %w", err)
		}

		roots, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("kafka: load system certificate pool: %w", err)
		}

		if roots == nil {
			roots = x509.NewCertPool()
		}

		if ok := roots.AppendCertsFromPEM(caPEM); !ok {
			return nil, errors.New(
				"kafka: append tls ca certificate",
			)
		}

		config.RootCAs = roots
	}

	if cfg.CertFile != "" {
		certificate, err := tls.LoadX509KeyPair(
			cfg.CertFile,
			cfg.KeyFile,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"kafka: load tls client certificate: %w",
				err,
			)
		}

		config.Certificates = []tls.Certificate{certificate}
	}

	return config, nil
}

func applySASL(
	config *sarama.Config,
	cfg SASLConfig,
) error {
	if !cfg.Enabled {
		return nil
	}

	config.Net.SASL.Enable = true
	config.Net.SASL.User = cfg.Username
	config.Net.SASL.Password = cfg.Password

	switch cfg.Mechanism {
	case "", SASLMechanismPlain:
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext

	case SASLMechanismSCRAMSHA256:
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		config.Net.SASL.SCRAMClientGeneratorFunc = newSCRAMSHA256Client

	case SASLMechanismSCRAMSHA512:
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		config.Net.SASL.SCRAMClientGeneratorFunc = newSCRAMSHA512Client

	default:
		return fmt.Errorf(
			"kafka: unsupported sasl mechanism %q",
			cfg.Mechanism,
		)
	}

	return nil
}
