package interview

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	applicationdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
	interviewdomain "github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
)

type ScheduleInterviewInput struct {
	ApplicationID      uuid.UUID
	InterviewType      string
	ScheduledStartTime time.Time
	ScheduledEndTime   time.Time
	Participants       []interviewdomain.Participant
	ActorID            uuid.UUID
	RequireOnlineRoom  bool
}

func (s *Service) ScheduleInterview(ctx context.Context, input ScheduleInterviewInput) (uuid.UUID, error) {
	if input.ApplicationID == uuid.Nil {
		return uuid.Nil, apperr.InvalidArgument("empty_application_id", "application ID is required", nil)
	}
	if input.ActorID == uuid.Nil {
		return uuid.Nil, apperr.InvalidArgument("empty_actor_id", "actor ID is required", nil)
	}

	if err := s.authPort.CanManageInterview(ctx, input.ActorID, input.ApplicationID); err != nil {
		return uuid.Nil, err
	}

	stage, err := s.appChecker.GetStage(ctx, input.ApplicationID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("check application stage: %w", err)
	}

	if stage != applicationdomain.StageInterview && stage != applicationdomain.StageScreening {
		return uuid.Nil, apperr.Conflict(
			"invalid_stage",
			fmt.Sprintf("cannot schedule interview for application in stage %s", stage),
			nil,
		)
	}

	now := s.clock.Now().UTC()
	interviewID := s.idGen.New()

	roomURL := ""
	if input.RequireOnlineRoom && s.conferenceSvc != nil {
		url, err := s.conferenceSvc.CreateRoom(ctx, fmt.Sprintf("interview-%s", interviewID))
		if err != nil {
			return uuid.Nil, fmt.Errorf("create conference room: %w", err)
		}
		roomURL = url
	}

	it, err := interviewdomain.Schedule(
		interviewID,
		input.ApplicationID,
		input.InterviewType,
		input.ScheduledStartTime,
		input.ScheduledEndTime,
		input.Participants,
		roomURL,
		now,
	)
	if err != nil {
		return uuid.Nil, err
	}

	payload := interviewdomain.InterviewScheduledPayload{
		InterviewID:   interviewID,
		ApplicationID: input.ApplicationID,
		InterviewType: it.InterviewType(),
		StartTime:     it.ScheduledStartTime(),
		EndTime:       it.ScheduledEndTime(),
		Participants:  it.Participants(),
	}

	version := it.Version()
	eventMsg, err := s.eventFactory.Create(
		interviewdomain.EventTypeInterviewScheduled,
		interviewdomain.AggregateTypeInterview,
		interviewID,
		&version,
		payload,
		now,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create interview scheduled event: %w", err)
	}

	err = s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.interviewRepo.Create(txCtx, it); err != nil {
			return fmt.Errorf("save interview: %w", err)
		}
		if err := s.outboxRepo.Save(txCtx, eventMsg); err != nil {
			return fmt.Errorf("save outbox event: %w", err)
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	return interviewID, nil
}

