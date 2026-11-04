package migration

import "fmt"

type Command string

const (
	CommandUp      Command = "up"
	CommandStatus  Command = "status"
	CommandVersion Command = "version"
)

func ParseCommand(value string) (Command, error) {
	command := Command(value)

	if err := command.Validate(); err != nil {
		return "", err
	}

	return command, nil
}

func (c Command) Validate() error {
	switch c {
	case CommandUp, CommandStatus, CommandVersion:
		return nil
	default:
		return fmt.Errorf("unsupported migration command: %q", c)
	}
}
