package redis

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	server := startRESPServer(t)
	host, portString, err := net.SplitHostPort(server.Address())
	require.NoError(t, err)
	port, err := strconv.Atoi(portString)
	require.NoError(t, err)

	cfg := validConfig()
	cfg.Connection = ConnectionConfig{
		Host:     host,
		Port:     port,
		Database: 2,
		Username: "application",
		Password: "secret",
	}
	cfg.DialTimeout = 2 * time.Second
	cfg.ReadTimeout = 1500 * time.Millisecond
	cfg.WriteTimeout = 1200 * time.Millisecond
	cfg.PoolSize = 4
	cfg.MinIdleConns = 0
	cfg.PoolTimeout = 1100 * time.Millisecond
	cfg.ConnMaxIdleTime = 3 * time.Minute
	cfg.ConnMaxLifetime = 5 * time.Minute
	cfg.MaxRetries = 1
	cfg.MinRetryBackoff = 2 * time.Millisecond
	cfg.MaxRetryBackoff = 10 * time.Millisecond

	client, err := New(context.Background(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	require.NotNil(t, client.Client())
	require.NotNil(t, client.Stats())
	require.NoError(t, client.Ping(context.Background()))

	options := client.Client().Options()
	require.Equal(t, cfg.Connection.Address(), options.Addr)
	require.Equal(t, cfg.Connection.Username, options.Username)
	require.Equal(t, cfg.Connection.Password, options.Password)
	require.Equal(t, cfg.Connection.Database, options.DB)
	require.Equal(t, cfg.DialTimeout, options.DialTimeout)
	require.Equal(t, cfg.ReadTimeout, options.ReadTimeout)
	require.Equal(t, cfg.WriteTimeout, options.WriteTimeout)
	require.Equal(t, cfg.PoolSize, options.PoolSize)
	require.Equal(t, cfg.MinIdleConns, options.MinIdleConns)
	require.Equal(t, cfg.PoolTimeout, options.PoolTimeout)
	require.Equal(t, cfg.ConnMaxIdleTime, options.ConnMaxIdleTime)
	require.Equal(t, cfg.ConnMaxLifetime, options.ConnMaxLifetime)
	require.Equal(t, cfg.MaxRetries, options.MaxRetries)
	require.Equal(t, cfg.MinRetryBackoff, options.MinRetryBackoff)
	require.Equal(t, cfg.MaxRetryBackoff, options.MaxRetryBackoff)

	require.Eventually(t, func() bool {
		commands := server.Commands()
		return containsCommand(commands, "AUTH") &&
			containsCommand(commands, "SELECT") &&
			containsCommand(commands, "PING")
	}, time.Second, 10*time.Millisecond)
}

func TestNewRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	client, err := New(context.Background(), Config{})

	require.Nil(t, client)
	require.EqualError(t, err, "validate redis config: redis host is required")
}

func TestNewReturnsPingError(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	host, portString, err := net.SplitHostPort(address)
	require.NoError(t, err)
	port, err := strconv.Atoi(portString)
	require.NoError(t, err)

	cfg := validConfig()
	cfg.Connection.Host = host
	cfg.Connection.Port = port
	cfg.DialTimeout = 100 * time.Millisecond
	cfg.MaxRetries = -1

	client, err := New(context.Background(), cfg)

	require.Nil(t, client)
	require.ErrorContains(t, err, "ping redis after create: ping redis:")
}

func TestNewHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client, err := New(ctx, validConfig())

	require.Nil(t, client)
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "ping redis after create: ping redis:")
}

func TestClientLifecycle(t *testing.T) {
	t.Parallel()

	server := startRESPServer(t)
	host, portString, err := net.SplitHostPort(server.Address())
	require.NoError(t, err)
	port, err := strconv.Atoi(portString)
	require.NoError(t, err)

	cfg := validConfig()
	cfg.Connection.Host = host
	cfg.Connection.Port = port
	cfg.MinIdleConns = 0

	client, err := New(context.Background(), cfg)
	require.NoError(t, err)
	require.NoError(t, client.Close())

	require.Nil(t, client.Client())
	require.Nil(t, client.Stats())
	require.ErrorIs(t, client.Ping(context.Background()), ErrClosed)
	require.NoError(t, client.Close())
}

func TestNilClient(t *testing.T) {
	t.Parallel()

	var client *Client

	require.Nil(t, client.Client())
	require.Nil(t, client.Stats())
	require.ErrorIs(t, client.Ping(context.Background()), ErrClosed)
	require.NoError(t, client.Close())
}

type respServer struct {
	listener net.Listener

	mu       sync.Mutex
	commands []string
}

func startRESPServer(t *testing.T) *respServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &respServer{listener: listener}
	t.Cleanup(func() { require.NoError(t, server.Close()) })

	go server.serve()

	return server
}

func (s *respServer) Address() string {
	return s.listener.Addr().String()
}

func (s *respServer) Commands() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.commands...)
}

func (s *respServer) Close() error {
	err := s.listener.Close()
	if err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}

	return nil
}

func (s *respServer) serve() {
	for {
		connection, err := s.listener.Accept()
		if err != nil {
			return
		}

		go s.handle(connection)
	}
}

func (s *respServer) handle(connection net.Conn) {
	defer func() { _ = connection.Close() }()

	reader := bufio.NewReader(connection)
	for {
		args, err := readRESPCommand(reader)
		if err != nil {
			return
		}
		if len(args) == 0 {
			continue
		}

		command := strings.ToUpper(args[0])
		s.mu.Lock()
		s.commands = append(s.commands, command)
		s.mu.Unlock()

		var response string
		switch command {
		case "HELLO":
			response = "-ERR unknown command 'hello'\r\n"
		case "PING":
			response = "+PONG\r\n"
		default:
			response = "+OK\r\n"
		}

		if _, err := io.WriteString(connection, response); err != nil {
			return
		}
	}
}

func readRESPCommand(reader *bufio.Reader) ([]string, error) {
	header, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	header = strings.TrimSuffix(strings.TrimSuffix(header, "\n"), "\r")
	if !strings.HasPrefix(header, "*") {
		return nil, fmt.Errorf("expected RESP array, got %q", header)
	}

	argumentCount, err := strconv.Atoi(strings.TrimPrefix(header, "*"))
	if err != nil {
		return nil, fmt.Errorf("parse RESP argument count: %w", err)
	}

	arguments := make([]string, 0, argumentCount)
	for range argumentCount {
		lengthHeader, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		lengthHeader = strings.TrimSuffix(strings.TrimSuffix(lengthHeader, "\n"), "\r")
		if !strings.HasPrefix(lengthHeader, "$") {
			return nil, fmt.Errorf("expected RESP bulk string, got %q", lengthHeader)
		}

		length, err := strconv.Atoi(strings.TrimPrefix(lengthHeader, "$"))
		if err != nil {
			return nil, fmt.Errorf("parse RESP bulk string length: %w", err)
		}

		value := make([]byte, length)
		if _, err := io.ReadFull(reader, value); err != nil {
			return nil, err
		}
		if _, err := reader.Discard(2); err != nil {
			return nil, err
		}

		arguments = append(arguments, string(value))
	}

	return arguments, nil
}

func containsCommand(commands []string, target string) bool {
	for _, command := range commands {
		if command == target {
			return true
		}
	}

	return false
}
