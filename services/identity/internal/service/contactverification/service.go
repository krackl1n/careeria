package contactverification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	emailchallengedomain "github.com/krackl1n/careeria/services/identity/internal/domain/emailverificationchallenge"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	phonechallengedomain "github.com/krackl1n/careeria/services/identity/internal/domain/phoneverificationchallenge"
)

type EmailChallengeRepository interface {
	RevokeActiveByEmailID(context.Context, identitydomain.EmailID, time.Time) (int64, error)
	Create(context.Context, *emailchallengedomain.EmailVerificationChallenge) error
	GetForUpdate(context.Context, emailchallengedomain.ID) (*emailchallengedomain.EmailVerificationChallenge, error)
	Update(context.Context, *emailchallengedomain.EmailVerificationChallenge) error
}

type PhoneChallengeRepository interface {
	RevokeActiveByPhoneID(context.Context, identitydomain.PhoneID, time.Time) (int64, error)
	Create(context.Context, *phonechallengedomain.PhoneVerificationChallenge) error
	GetForUpdate(context.Context, phonechallengedomain.ID) (*phonechallengedomain.PhoneVerificationChallenge, error)
	Update(context.Context, *phonechallengedomain.PhoneVerificationChallenge) error
}

type CodeGenerator interface {
	Generate() (string, error)
}

type TokenDigester interface {
	Digest(string) string
}

type IDGenerator interface {
	New() uuid.UUID
}

type OutboxWriter interface {
	Create(context.Context, eventdomain.Message) error
}

type issuedChallenge struct {
	ChallengeID uuid.UUID
	ExpiresAt   time.Time
}

type Service struct {
	cfg             Config
	emailChallenges EmailChallengeRepository
	phoneChallenges PhoneChallengeRepository
	outbox          OutboxWriter
	codes           CodeGenerator
	digester        TokenDigester
	ids             IDGenerator
	events          eventdomain.Factory
}

func NewService(
	cfg Config,
	emailChallenges EmailChallengeRepository,
	phoneChallenges PhoneChallengeRepository,
	outbox OutboxWriter,
	codes CodeGenerator,
	digester TokenDigester,
	ids IDGenerator,
) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate contact verification config: %w", err)
	}
	if emailChallenges == nil {
		return nil, errors.New("create contact verification service: email challenge repository is required")
	}
	if phoneChallenges == nil {
		return nil, errors.New("create contact verification service: phone challenge repository is required")
	}
	if outbox == nil {
		return nil, errors.New("create contact verification service: outbox writer is required")
	}
	if codes == nil {
		return nil, errors.New("create contact verification service: code generator is required")
	}
	if digester == nil {
		return nil, errors.New("create contact verification service: token digester is required")
	}
	if ids == nil {
		return nil, errors.New("create contact verification service: ID generator is required")
	}

	return &Service{
		cfg:             cfg,
		emailChallenges: emailChallenges,
		phoneChallenges: phoneChallenges,
		outbox:          outbox,
		codes:           codes,
		digester:        digester,
		ids:             ids,
		events:          eventdomain.NewFactory(ids),
	}, nil
}
