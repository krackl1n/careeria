package kafka

import (
	"crypto/sha256"
	"crypto/sha512"

	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"
)

type scramClient struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

func newSCRAMSHA256Client() sarama.SCRAMClient {
	return &scramClient{
		HashGeneratorFcn: scram.HashGeneratorFcn(sha256.New),
	}
}

func newSCRAMSHA512Client() sarama.SCRAMClient {
	return &scramClient{
		HashGeneratorFcn: scram.HashGeneratorFcn(sha512.New),
	}
}

func (c *scramClient) Begin(
	username,
	password,
	authzID string,
) error {
	client, err := c.HashGeneratorFcn.NewClient(
		username,
		password,
		authzID,
	)
	if err != nil {
		return err
	}

	c.Client = client
	c.ClientConversation = client.NewConversation()

	return nil
}

func (c *scramClient) Step(
	challenge string,
) (string, error) {
	return c.ClientConversation.Step(challenge)
}

func (c *scramClient) Done() bool {
	return c.ClientConversation.Done()
}
