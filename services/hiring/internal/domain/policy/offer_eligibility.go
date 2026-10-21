package policy

import (
	"fmt"

	"github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/interview"
)

type OfferEligibilityPolicy struct{}

func NewOfferEligibilityPolicy() *OfferEligibilityPolicy {
	return &OfferEligibilityPolicy{}
}

func (p *OfferEligibilityPolicy) CanInitiateOffer(
	app *application.JobApplication,
	interviews []interview.InterviewSummary,
) error {
	if app == nil {
		return ErrNilApplication
	}

	if app.CurrentStage != application.StageInterview {
		return fmt.Errorf("%w: application is in stage %s", ErrInvalidStageForOffer, app.CurrentStage)
	}

	hasCompletedWithFeedback := false
	for _, it := range interviews {
		if it.Status == interview.StatusCompleted && it.HasFeedback {
			hasCompletedWithFeedback = true
			break
		}
	}

	if !hasCompletedWithFeedback {
		return ErrMissingCompletedInterview
	}

	return nil
}
