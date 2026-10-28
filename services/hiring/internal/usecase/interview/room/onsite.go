package room

import (
	"context"
	"errors"
	"fmt"
	"strings"

	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
)

var ErrEmptyOfficeLocation = errors.New("office location is required for onsite interview")

type OnsiteStrategy struct {
	defaultOfficeAddress string
}

func NewOnsiteStrategy(defaultOfficeAddress string) *OnsiteStrategy {
	return &OnsiteStrategy{
		defaultOfficeAddress: strings.TrimSpace(defaultOfficeAddress),
	}
}

func (s *OnsiteStrategy) Format() interviewdomain.MeetingFormat {
	return interviewdomain.FormatOnsite
}

func (s *OnsiteStrategy) Provision(_ context.Context, req ProvisionRequest) (*ProvisionResult, error) {
	location := strings.TrimSpace(req.Location)
	if location == "" {
		location = s.defaultOfficeAddress
	}

	if location == "" {
		return nil, ErrEmptyOfficeLocation
	}

	return &ProvisionResult{
		MeetingURL: location,
		Metadata: map[string]string{
			"provider": "onsite",
			"location": location,
		},
	}, nil
}

func (s *OnsiteStrategy) GenerateAccess(_ context.Context, req AccessRequest) (*AccessResult, error) {
	instructions := fmt.Sprintf(
		"Please arrive at %s 10 minutes before the scheduled time with an ID document for the reception desk.",
		req.MeetingURL,
	)

	return &AccessResult{
		JoinURL:      req.MeetingURL,
		Instructions: instructions,
	}, nil
}

func (s *OnsiteStrategy) Teardown(_ context.Context, _ TeardownRequest) error {
	return nil
}
