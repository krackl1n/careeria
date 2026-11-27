package vacancy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

type CreateDraftInput struct {
	RequestID uuid.UUID
	CompanyID uuid.UUID
	// AuthorID comes from the authenticated transport context.
	AuthorID uuid.UUID
	Details  domain.Details
}

func (s *Service) CreateDraft(ctx context.Context, input CreateDraftInput) (*domain.Vacancy, error) {
	if input.RequestID == uuid.Nil || input.CompanyID == uuid.Nil || input.AuthorID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal create input: %w", err)
	}
	digest := sha256.Sum256(data)
	hash := hex.EncodeToString(digest[:])
	existing, storedHash, err := s.repository.GetByCreateRequest(ctx, input.RequestID)
	if err == nil {
		return createResult(existing, input, storedHash, hash)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("check create request: %w", err)
	}

	entity, err := domain.NewDraft(s.ids.New(), input.CompanyID, input.AuthorID, input.Details, s.clock.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		return nil, err
	}
	err = s.transactions.Do(ctx, func(txCtx context.Context) error {
		if err := s.repository.Create(txCtx, entity, input.RequestID, hash); err != nil {
			return err
		}
		return s.recordEvent(txCtx, "vacancy.draft_created", entity.Snapshot())
	})
	if errors.Is(err, domain.ErrConflict) {
		existing, storedHash, lookupErr := s.repository.GetByCreateRequest(ctx, input.RequestID)
		if lookupErr == nil {
			return createResult(existing, input, storedHash, hash)
		}
		if !errors.Is(lookupErr, domain.ErrNotFound) {
			return nil, fmt.Errorf("resolve create conflict: %w", lookupErr)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("create vacancy: %w", err)
	}
	return createResult(entity, input, hash, hash)
}

func createResult(entity *domain.Vacancy, input CreateDraftInput, storedHash, hash string) (*domain.Vacancy, error) {
	if storedHash != hash {
		return nil, ErrIdempotencyConflict
	}
	state := entity.Snapshot()
	return domain.NewDraft(state.ID, state.CompanyID, state.AuthorID, input.Details, state.CreatedAt.UTC())
}
