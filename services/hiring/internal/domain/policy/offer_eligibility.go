package policy

import (
	"fmt"

	"github.com/krackl1n/careeria/services/hiring/internal/domain/application"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/apperr"
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
		return apperr.InvalidArgument("nil_application", "application is required", nil)
	}

	if app.CurrentStage() != application.StageInterview {
		return apperr.Conflict(
			"invalid_stage_for_offer",
			fmt.Sprintf("cannot initiate offer for application in stage %s; must be in %s",
				app.CurrentStage(), application.StageInterview),
			nil,
		)
	}

	hasCompletedWithFeedback := false
	for _, it := range interviews {
		if it.Status == interview.StatusCompleted && it.HasFeedback {
			hasCompletedWithFeedback = true
			break
		}
	}

	if !hasCompletedWithFeedback {
		return apperr.Conflict(
			"missing_completed_interview",
			"cannot initiate offer: at least one interview must be completed with feedback",
			nil,
		)
	}

	return nil
}


