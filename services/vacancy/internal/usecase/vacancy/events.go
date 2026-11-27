package vacancy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/krackl1n/careeria/services/vacancy/internal/domain/event"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
)

func (s *Service) recordEvent(ctx context.Context, kind string, state domain.State) error {
	payload, err := json.Marshal(struct {
		VacancyID string        `json:"vacancy_id"`
		CompanyID string        `json:"company_id"`
		Version   int64         `json:"version"`
		Status    domain.Status `json:"status"`
	}{state.ID.String(), state.CompanyID.String(), state.Version, state.Status})
	if err != nil {
		return fmt.Errorf("marshal vacancy event: %w", err)
	}
	message := event.Message{
		ID: s.ids.New(), EventID: s.ids.New(), AggregateID: state.ID,
		AggregateVersion: state.Version, EventType: kind, Payload: payload, OccurredAt: state.UpdatedAt,
	}
	if err := s.outbox.Create(ctx, message); err != nil {
		return fmt.Errorf("write vacancy event: %w", err)
	}
	return nil
}
